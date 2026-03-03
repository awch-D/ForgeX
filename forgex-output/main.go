package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"todo-cli/todo"
)

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: todo <command> [arguments]

Commands:
  add <title>    Add a new todo
  list           List all todos
  done <id>      Mark a todo as done
  delete <id>    Delete a todo
`)
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	store := todo.NewStore()
	if err := store.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "add":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Error: title is required")
			fmt.Fprintln(os.Stderr, "Usage: todo add <title>")
			os.Exit(1)
		}
		title := strings.Join(os.Args[2:], " ")
		t := store.Add(title)
		if err := store.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Added: %s\n", t)

	case "list":
		if len(store.Todos) == 0 {
			fmt.Println("No todos yet. Add one with: todo add <title>")
			return
		}
		pending, done := 0, 0
		for _, t := range store.Todos {
			fmt.Println(t)
			if t.Done {
				done++
			} else {
				pending++
			}
		}
		fmt.Printf("\n%d pending, %d done, %d total\n", pending, done, len(store.Todos))

	case "done":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Error: id is required")
			fmt.Fprintln(os.Stderr, "Usage: todo done <id>")
			os.Exit(1)
		}
		id, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid id %q\n", os.Args[2])
			os.Exit(1)
		}
		if err := store.Done(id); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := store.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Todo %d marked as done\n", id)

	case "delete":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Error: id is required")
			fmt.Fprintln(os.Stderr, "Usage: todo delete <id>")
			os.Exit(1)
		}
		id, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid id %q\n", os.Args[2])
			os.Exit(1)
		}
		if err := store.Delete(id); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := store.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Todo %d deleted\n", id)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		usage()
	}
}
