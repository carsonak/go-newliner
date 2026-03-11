package a

import "fmt"

// --- Rule 1: Closing braces need a blank line ---

func rule1_missing_blank() {
	if true {
		fmt.Println("inside")
	} // want "closing brace should be followed by a blank line"
	fmt.Println("after if")
}

func rule1_ok() {
	if true {
		fmt.Println("inside")
	}

	fmt.Println("after if")
}

func rule1_for_missing_blank() {
	for i := 0; i < 1; i++ {
		fmt.Println(i)
	} // want "closing brace should be followed by a blank line"
	fmt.Println("after for")
}

func rule1_for_ok() {
	for i := 0; i < 1; i++ {
		fmt.Println(i)
	}

	fmt.Println("after for")
}

// Rule 1 Exception B: next non-ws char is }
func rule1_exceptionB_closing() {
	if true {
		if true {
			fmt.Println("nested")
		}
	}
}

// Rule 1 Exception B: for inside func, next is }
func rule1_exceptionB_for() {
	for i := 0; i < 1; i++ {
		fmt.Println(i)
	}
}

// --- Rule 2: Declarations need a blank line ---

func rule2_missing_blank() {
	x := 1 // want "declaration should be followed by a blank line"
	fmt.Println(x)
}

func rule2_ok() {
	x := 1

	fmt.Println(x)
}

// Rule 2: contiguous declarations - only last needs blank
func rule2_contiguous_missing() {
	x := 1
	y := 2 // want "declaration should be followed by a blank line"
	fmt.Println(x, y)
}

// Rule 2 Exception A: error check right after
func rule2_exceptionA() {
	x, err := fmt.Println("hello")
	if err != nil {
		return
	}

	_ = x
}

// Rule 2 Exception A: custom error variable name
func rule2_exceptionA_custom_errvar() {
	data, readErr := fmt.Println("hello")
	if readErr != nil {
		return
	}

	_ = data
}

// --- Rule 3: Go statements need a blank line ---

func rule3_missing_blank() {
	go func() {}() // want "go statement should be followed by a blank line"
	fmt.Println("after go")
}

func rule3_ok() {
	go func() {}()

	fmt.Println("after go")
}

// Rule 3 Exception A: next non-ws is }
func rule3_exceptionA() {
	go func() {}()
}

// Rule 3: contiguous go stmts
func rule3_contiguous_missing() {
	go func() {}()
	go func() {}() // want "go statement should be followed by a blank line"
	fmt.Println("after")
}

// --- Rule 1 Exception A: defer cleanup ---

func rule1_exceptionA_defer() {
	f, err := openSomething()
	if err != nil {
		return
	}
	defer f.Close()

	fmt.Println(f)
}

// Rule 1 Exception A: defer cleanup with custom error variable name
func rule1_exceptionA_defer_custom_errvar() {
	f, openErr := openSomething()
	if openErr != nil {
		return
	}
	defer f.Close()

	fmt.Println(f)
}

func openSomething() (interface{ Close() error }, error) {
	return nil, nil
}
