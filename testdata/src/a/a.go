package a

import (
	"fmt"
	"log"
)

type Person struct {
	Name string
	Age  int
}

func show(f func(format string, v ...interface{}), v interface{}) {
	f("show %v", v)
}

func wrappedLog(v interface{}) {
	log.Printf("WrappedLog: %d %v", 100, v)
}

func wrappedLog2(v interface{}) {
	v = 500
	log.Printf("WrappedLog: %d %v", 100, v)
}

func AdultOnly(people []Person) bool {
	xxx := 100
	for _, person := range people {
		if person.Age < 18 {
			fmt.Printf("Found not adult: %v\n", person) // OK
			log.Printf("Found not adult: %v\n", person) // want "NG"

			p := &person
			log.Printf("Found not adult: %v\n", p) // (wish) want "NG"

			f := log.Printf
			f("test: %v\n", person) // want "NG"

			g := log.Printf
			if person.Age == 5 {
				g = log.Printf
			}
			g("test: %v\n", person) // OK

			wrappedLog(person)  // want "NG"
			wrappedLog2(person) // (wish) OK

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
