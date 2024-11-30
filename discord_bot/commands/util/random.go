package util

import "math/rand"

func GetRandom(min, max int) int {
	return rand.Intn(max-min+1) + min
}

// UniqueRand generates unique random integers.
type UniqueRand struct {
	generated map[int]bool
}

func NewUniqueRand() *UniqueRand {
	return &UniqueRand{
		generated: make(map[int]bool),
	}
}

func (u *UniqueRand) Intn(min, max int) int {
	for {
		i := GetRandom(min, max)
		if !u.generated[i] {
			u.generated[i] = true
			return i
		}
	}
}
