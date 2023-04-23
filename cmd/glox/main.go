package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/vikblom/glox"
)

func runMain() error {
	sc := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("> ")
		if !sc.Scan() {
			break
		}
		lexer := glox.NewScanner(sc.Bytes())
		for {
			tok := lexer.Scan()
			if tok.Kind == glox.EOF {
				break
			}
			fmt.Printf("%v\n", tok)
		}
	}
	return nil
}

func main() {
	err := runMain()
	if err != nil {
		fmt.Printf("glox failed: %s", err)
		os.Exit(1)
	}
}
