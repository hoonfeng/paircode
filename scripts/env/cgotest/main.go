package main

/*
#include <stdio.h>
static int add42(int x) { return x + 42; }
*/
import "C"
import "fmt"

func main() {
	fmt.Println("cgo 可用 →", int(C.add42(0)))
}
