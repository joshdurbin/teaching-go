package main

import (
	"flag"
	"fmt"
	"math/rand"
	"time"
)

// LinkedListNode represents a node in a standard linked list
type LinkedListNode struct {
	value int
	next  *LinkedListNode
}

// LinkedList represents a simple singly linked list
type LinkedList struct {
	head *LinkedListNode
	size int
}

// Insert adds a value to the linked list (unsorted)
func (ll *LinkedList) Insert(value int) {
	newNode := &LinkedListNode{value: value, next: ll.head}
	ll.head = newNode
	ll.size++
}

// Find searches for a value in the linked list
func (ll *LinkedList) Find(value int) bool {
	current := ll.head
	for current != nil {
		if current.value == value {
			return true
		}
		current = current.next
	}
	return false
}

// SkipListNode represents a node in a skip list with multiple forward pointers
type SkipListNode struct {
	value   int
	forward []*SkipListNode
}

// SkipList represents a probabilistic data structure for fast search
type SkipList struct {
	head      *SkipListNode
	maxLevel  int
	level     int
	size      int
	rng       *rand.Rand
}

// NewSkipList creates a new skip list with specified max levels
func NewSkipList(maxLevel int) *SkipList {
	return &SkipList{
		head:     &SkipListNode{value: -1, forward: make([]*SkipListNode, maxLevel)},
		maxLevel: maxLevel,
		level:    0,
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// randomLevel generates a random level for a new node
func (sl *SkipList) randomLevel() int {
	level := 0
	for level < sl.maxLevel-1 && sl.rng.Float32() < 0.5 {
		level++
	}
	return level
}

// Insert adds a value to the skip list
func (sl *SkipList) Insert(value int) {
	update := make([]*SkipListNode, sl.maxLevel)
	current := sl.head

	// Find the position to insert
	for i := sl.level; i >= 0; i-- {
		for current.forward[i] != nil && current.forward[i].value < value {
			current = current.forward[i]
		}
		update[i] = current
	}

	// Generate random level for new node
	newLevel := sl.randomLevel()
	if newLevel > sl.level {
		for i := sl.level + 1; i <= newLevel; i++ {
			update[i] = sl.head
		}
		sl.level = newLevel
	}

	// Create new node and update pointers
	newNode := &SkipListNode{
		value:   value,
		forward: make([]*SkipListNode, newLevel+1),
	}

	for i := 0; i <= newLevel; i++ {
		newNode.forward[i] = update[i].forward[i]
		update[i].forward[i] = newNode
	}

	sl.size++
}

// Find searches for a value in the skip list
func (sl *SkipList) Find(value int) bool {
	current := sl.head

	for i := sl.level; i >= 0; i-- {
		for current.forward[i] != nil && current.forward[i].value < value {
			current = current.forward[i]
		}
	}

	current = current.forward[0]
	return current != nil && current.value == value
}

func main() {
	// Command line flags
	numElements := flag.Int("elements", 1000000, "Number of elements to insert")
	numSearches := flag.Int("searches", 10000, "Number of search operations to perform")
	maxLevel := flag.Int("maxlevel", 16, "Maximum level for skip list")
	seed := flag.Int64("seed", time.Now().UnixNano(), "Random seed for reproducibility")
	flag.Parse()

	fmt.Printf("Data Structure Performance Comparison\n")
	fmt.Printf("=====================================\n")
	fmt.Printf("Elements: %d\n", *numElements)
	fmt.Printf("Searches: %d\n", *numSearches)
	fmt.Printf("Skip List Max Level: %d\n\n", *maxLevel)

	rng := rand.New(rand.NewSource(*seed))

	// Generate random data to insert
	fmt.Println("Generating random data...")
	data := make([]int, *numElements)
	for i := 0; i < *numElements; i++ {
		data[i] = rng.Intn(*numElements * 10)
	}

	// Generate random search queries
	searchQueries := make([]int, *numSearches)
	for i := 0; i < *numSearches; i++ {
		searchQueries[i] = rng.Intn(*numElements * 10)
	}

	// Benchmark Linked List
	fmt.Println("Building Linked List...")
	ll := &LinkedList{}
	startInsert := time.Now()
	for _, value := range data {
		ll.Insert(value)
	}
	llInsertDuration := time.Since(startInsert)

	fmt.Printf("Linked List insert time: %v\n", llInsertDuration)
	fmt.Printf("Linked List size: %d\n", ll.size)

	// Benchmark Skip List
	fmt.Println("\nBuilding Skip List...")
	sl := NewSkipList(*maxLevel)
	startInsert = time.Now()
	for _, value := range data {
		sl.Insert(value)
	}
	slInsertDuration := time.Since(startInsert)

	fmt.Printf("Skip List insert time: %v\n", slInsertDuration)
	fmt.Printf("Skip List size: %d\n", sl.size)
	fmt.Printf("Skip List actual levels: %d\n", sl.level+1)

	// Benchmark Linked List Search
	fmt.Println("\nSearching Linked List...")
	llFoundCount := 0
	startSearch := time.Now()
	for _, query := range searchQueries {
		if ll.Find(query) {
			llFoundCount++
		}
	}
	llSearchDuration := time.Since(startSearch)

	fmt.Printf("Linked List search time: %v\n", llSearchDuration)
	fmt.Printf("Linked List found: %d/%d\n", llFoundCount, *numSearches)
	fmt.Printf("Linked List avg per search: %v\n", llSearchDuration/time.Duration(*numSearches))

	// Benchmark Skip List Search
	fmt.Println("\nSearching Skip List...")
	slFoundCount := 0
	startSearch = time.Now()
	for _, query := range searchQueries {
		if sl.Find(query) {
			slFoundCount++
		}
	}
	slSearchDuration := time.Since(startSearch)

	fmt.Printf("Skip List search time: %v\n", slSearchDuration)
	fmt.Printf("Skip List found: %d/%d\n", slFoundCount, *numSearches)
	fmt.Printf("Skip List avg per search: %v\n", slSearchDuration/time.Duration(*numSearches))

	// Summary
	fmt.Println("\n" + "=====Summary=====")
	fmt.Printf("Insert speedup (Skip List vs Linked List): %.2fx\n",
		float64(llInsertDuration)/float64(slInsertDuration))
	fmt.Printf("Search speedup (Skip List vs Linked List): %.2fx\n",
		float64(llSearchDuration)/float64(slSearchDuration))
}
