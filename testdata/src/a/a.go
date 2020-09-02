package a

import (
	"fmt"
	"log"
)

type Sex int

const (
	Man Sex = iota
	Woman
	Otherwise
)

// HIDE FROM log.Printf log.Panicf
type Person struct {
	Name string
	Sex  Sex
	Age  int
}

func show(f func(format string, v ...interface{}), v interface{}) {
	f("show %v", v)
}

func AdultOnly(people []Person) bool {
	xxx := 100
	for _, person := range people {
		if person.Age < 18 {
			fmt.Printf("Found not adult: %v\n", person) // OK
			log.Printf("Found not adult: %v\n", person) // want "NG"

			p := &person
			log.Printf("Found not adult: %v\n", p) // want "NG"

			f := log.Printf
			f("test: %v\n", person) // want "NG"

			g := log.Printf
			if person.Age == 5 {
				g = log.Printf
			}
			g("test: %v\n", person) // OK

			show(log.Printf, person) // OK

			return false
		}
		if person.Age < 0 {
			log.Printf("Panic: %v\n", xxx)             // OK
			log.Panicf("impossible human: %v", person) // want "NG"
		}
	}
	return true
}
