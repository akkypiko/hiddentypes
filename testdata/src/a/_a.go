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

// HIDE FROM log.Printf log.Panic
type Person struct {
	Name string
	Sex  Sex
	Age  int
}

func (p *Person) Error() string { // want "NG"
	return "hoge"
}

func (p *Person) Say(s string) {
	fmt.Printf("%s says %s.\n", p.Name, s)
	log.Printf("LOG: %s says %s\n", p.Name)
}

func AdultOnly(people []Person) bool {
	for _, person := range people {
		if person.Age < 18 {
			log.Printf("Found not adult: %v\n", person) // want "NG"
			return false
		}
	}
	return true
}

func mylog(v interface{}) {
	log.Print("MYLOG: ")
	mylogvalue(v)
}

func mylogvalue(v interface{}) {
	log.Printf("%v\n", v)
}

func ManOnly(people []Person) bool {
	for _, person := range people {
		if person.Sex != Man {
			mylog(person) // want "NG"
			return false
		}
	}
	return true
}

func panicByName(p *Person) {
	log.Panic("LOGNAME %s\n", p.Name) // want "NG"
}

func panicByNameEvil(p *Person) {
	name := p.Name
	log.Panic("LOGNAME %s\n", name) // OK but wish want "NG"
}

type MyError struct {
	criminal Person
}

func (m *MyError) Error() string { // want "NG"
	return m.criminal.Name
}

func main() {
	people := []Person{
		Person{"Alice", Woman, 20},
		Person{"Bob", Man, 53},
		Person{"Charlie", Man, 12},
		Person{"David", Man, 27},
	}

	if AdultOnly(people) {
		fmt.Printf("Drink Sake!")
	}

}
