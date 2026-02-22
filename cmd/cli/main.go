package main

import (
    "fmt"
    "os"

    "myproject/internal/user"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("usage: cli <command>")
        os.Exit(1)
    }

    switch os.Args[1] {
    case "list-users":
        // use the same store as the server
    case "create-user":
        // ...
    }
}
