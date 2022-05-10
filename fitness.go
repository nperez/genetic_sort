package genetic_sort

import (
//copier "github.com/jinzhu/copier"
)

// Fitness requirements for advancing to the survivors list is probably where
// the most of the tuning will need to go in order to evolve anything useful.

// Stage 0. Survival
//  The first filter is if any of the random permutations crash the VM (out of
//  bounds access, slot over/underflow, endless loops, Technically, a gene set
//  of all no-ops would survive
// Stage 1. Pre-sorting
//  The second filter is if any permutations don't mutate the elements of the set. We want to select for fitness that will shuffle values, but not destroy the original values. An all no-op set would survive
// Stage 2. Sort
//  The third stage is where permutations that preserve original values but change element order for at least two elements. An all no-op set would not survive and individual genes with all no-op instructions might not be selected due to dominant gene selection. At this stage, the "sortedness" of the input set is the primary test. http://people.csail.mit.edu/mip/papers/invs/paper.pdf
// Stage 3. Refinement
//  The fourth stage is refinement or optimization. Here we start selecting for smaller instruction counts, less loops, runtime duration

type Fitness struct {
	SurvivalMinimum  int
	PreSortedMinimum int
	SortedMinimum    int
}

// Fitness scores are simple [0,100] and each test is free to return any int in that set. Units are iterated

type FitnessScore struct {
	Survival  int
	PreSorted int
	Sorted    int
	Alive     bool
}

func NewFitness(survival int, presorted int, sorted int) *Fitness {
	return &Fitness{
		SurvivalMinimum:  survival,
		PreSortedMinimum: presorted,
		SortedMinimum:    sorted,
	}
}

func (f *Fitness) Process(report *GenerationReport) {
	//	fit := &FitnessScore{
	//		Survival:  Survival(report),
	//		PreSorted: PreSorted(report),
	//		Sorted:    Sorted(report),
	//	}
	//
	//	fit.Alive =
	//		fit.Survival >= f.SurvivalMinimum &&
	//			fit.PreSorted >= f.PreSortedMinimum &&
	//			fit.Sorted >= f.SortedMinimum
}

//func Survival(report *GenerationReport) int {
//	if r.Exception == nil {
//		return 100
//	}
//}

//func PreSorted(report *GenerationReport) int {
//
//	inMap := make(map[int]int)
//	outMap := make(map[int]int)
//
//	for g := 0; g < len(r.Input); g++ {
//		inMap[report.Input[g]]++
//		outMap[report.Output[g]]++
//	}
//
//	total, count := 0, 0
//	for k, v := range inMap {
//		total++
//		if _, ok := outMap[k]; ok {
//			count++
//		}
//	}
//	return int(float64(count) / total * 100)
//}

//func Sorted(report *GenerationReport) int {
//	var outputCopy []int
//	copier.Copy(outputCopy, report.Output)
//	inversions := merge_sort(outputCopy)
//	maxInversions := len(outputCopy) * (len(outputCopy) - 1) / 2
//	return (100 / maxInversions) * (maxInversions - inversions)
//}

//func merge(a []int, inversion0 int) int {
//
//	inversion1 := 0
//
//	var c []int
//	copier.Copy(c, a)
//
//	copyLeft := 0
//	copyRight := len(c) / 2
//	current := 0
//	high := len(c) - 1
//
//	for copyLeft <= midpoint-1 && copyRight <= high {
//		if c[copyLeft] <= c[copyRight] {
//			a[current] = c[copyLeft]
//			copyLeft++
//		} else {
//			a[current] = c[copyRight]
//			copyRight++
//			inversion1 += midpoint - copyLeft
//		}
//		current++
//	}
//
//	for copyLeft <= midpoint-1 {
//		a[current] = c[copyLeft]
//		current++
//		copyLeft++
//	}
//
//	return inversion0 + inversion1
//}

//func merge_sort(a []int) int {
//	inversions := 0
//	if len(a) > 1 {
//		mid := len(a) / 2
//		if mid >= 1<<11 {
//			reply := make(<-chan int, 0)
//			go func() {
//				reply <- merge_sort(a[mid:])
//			}()
//			inv1 := merge_sort(a[:mid])
//			inv2 := <-reply
//			return merge(a, inv1+inv2)
//		}
//	}
//	return inversions
//}
