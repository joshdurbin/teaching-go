package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// Counter defines the common contract for the counters
type Counter interface {
	IncrementBy(value int)
	DecrementBy(value int)
	Value() int
}

// This is a decorator pattern implementation that adds timing functionality and name to track any Counter implementation
// and keeps the total operation count and total time spent in operations and reduces code duplication
// within the counters themselves.
type TimedCounter struct {
	name        string
	delegate    Counter
	totalTimeNs atomic.Int64
	totalOps    atomic.Int64
}

func NewTimedCounter(name string, delegate Counter) *TimedCounter {
	return &TimedCounter{
		name:     name,
		delegate: delegate,
	}
}

func (c *TimedCounter) Name() string {
	return c.name
}

func (c *TimedCounter) IncrementBy(value int) {
	start := time.Now()
	c.totalOps.Add(1)
	c.totalTimeNs.Add(time.Since(start).Nanoseconds())
	c.delegate.IncrementBy(value)
}

func (c *TimedCounter) DecrementBy(value int) {
	start := time.Now()
	c.totalOps.Add(1)
	c.totalTimeNs.Add(time.Since(start).Nanoseconds())
	c.delegate.DecrementBy(value)
}

// Value retrieves the current value from the underlying counter
// Is not necessary to time or count these operations as they happen in the main thread after all operations are complete
func (c *TimedCounter) Value() int {
	val := c.delegate.Value()
	return val
}

func (c *TimedCounter) TotalTime() time.Duration {
	return time.Duration(c.totalTimeNs.Load())
}

func (c *TimedCounter) TotalOps() int64 {
	return c.totalOps.Load()
}

type MutexCounter struct {
	mu    sync.RWMutex
	count int
}

func (c *MutexCounter) IncrementBy(value int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count += value
}

func (c *MutexCounter) DecrementBy(value int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count -= value
}

func (c *MutexCounter) Value() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.count
}

type ThreadUnsafeCounter struct {
	count int
}

func (c *ThreadUnsafeCounter) IncrementBy(value int) {
	c.count += value
}

func (c *ThreadUnsafeCounter) DecrementBy(value int) {
	c.count -= value
}

func (c *ThreadUnsafeCounter) Value() int {
	return c.count
}

type AtomicIntCounter struct {
	count atomic.Int32
}

func (c *AtomicIntCounter) IncrementBy(value int) {
	c.count.Add(int32(value))
}

func (c *AtomicIntCounter) DecrementBy(value int) {
	c.count.Add(int32(-value))
}

func (c *AtomicIntCounter) Value() int {
	return int(c.count.Load())
}

type ChannelCounter struct {
	ctx            context.Context
	increments     chan int
	decrements     chan int
	valueRetrieval chan chan int
	count          int
}

func CreateAndRunChannelCounter(ctx context.Context) *ChannelCounter {
	c := &ChannelCounter{
		ctx:            ctx,
		increments:     make(chan int, 64),
		decrements:     make(chan int, 64),
		valueRetrieval: make(chan chan int),
	}
	go c.run()
	return c
}

func (c *ChannelCounter) run() {
	for {
		select {
		case v := <-c.increments:
			c.count += v
		case v := <-c.decrements:
			c.count -= v
		case reply := <-c.valueRetrieval:
			reply <- c.count
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *ChannelCounter) IncrementBy(value int) {
	select {
	case c.increments <- value:
	case <-c.ctx.Done():
	}
}

func (c *ChannelCounter) DecrementBy(value int) {
	select {
	case c.decrements <- value:
	case <-c.ctx.Done():
	}
}

func (c *ChannelCounter) Value() int {
	reply := make(chan int)
	select {
	case c.valueRetrieval <- reply:
		return <-reply
	case <-c.ctx.Done():
		return c.count
	}
}

func main() {

	numRoutines := flag.Int("routines", 100, "the number of routines to run")
	numLoopPerRoutine := flag.Int("loops", 10000, "the number of loops or iterations to run per routine")

	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	counters := []*TimedCounter{}
	counters = append(counters,
		NewTimedCounter("Mutex", &MutexCounter{}),
		NewTimedCounter("Unsafe", &ThreadUnsafeCounter{}),
		NewTimedCounter("AtomicInt", &AtomicIntCounter{}),
		NewTimedCounter("Channel and worker", CreateAndRunChannelCounter(ctx)))

	var wg sync.WaitGroup

	// iterate through the number of configured go routines to spin up
	for i := 0; i < *numRoutines; i++ {

		// place the async func into a wait group directly
		wg.Go(func() {

			// iterate through the number of loops per routine
			for i := 0; i < *numLoopPerRoutine; i++ {

				// check for context cancellation
				select {
				case <-ctx.Done():
					return
				default:
				}

				// randomly select an operation
				switch rand.Intn(2) + 1 {
				case 1:
					randValue := rand.Intn(5)
					for _, counter := range counters {
						counter.DecrementBy(randValue)
					}
				case 2:
					randValue := rand.Intn(5)
					for _, counter := range counters {
						counter.IncrementBy(randValue)
					}
				}
			}

		})
	}

	// block the main thread until all routines complete
	// if we don't do this, the main thread may exit before any of the routines start, honestly, and definitely before they complete
	wg.Wait()

	// range through the counters and get their final values and stats
	for _, counter := range counters {
		fmt.Printf("%s value is %d with a collective operation count of %v and processing time of %v\n", counter.Name(), counter.Value(), counter.TotalOps(), counter.TotalTime())
	}
}
