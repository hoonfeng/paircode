package main

import (
	"fmt"
	"os"

	"wb-ui/jsc"
)

func main() {
	interp := jsc.NewInterpreter()
	// Test for await...of parsing (uses sync iterator fallback)
	_, err := interp.RunJS(`
		async function testForAwaitOf() {
			var results = [];
			for await (const x of [10, 20, 30]) {
				results.push(x);
			}
			return JSON.stringify(results);
		}
		testForAwaitOf().then(function(v) {
			globalThis._testResult = v;
		});
	`)
	if err != nil {
		fmt.Printf("FAIL: %v\n", err)
		os.Exit(1)
	}
	result, err := interp.EvalJS(`globalThis._testResult`)
	if err != nil {
		fmt.Printf("FAIL: cannot read result: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("for await...of result: %s\n", result.ToString())
	if result.ToString() != `[10,20,30]` {
		fmt.Printf("FAIL: expected [10,20,30], got %s\n", result.ToString())
		os.Exit(1)
	}
	fmt.Println("PASS: for await...of works correctly!")
}
