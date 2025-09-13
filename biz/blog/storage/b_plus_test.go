package storage

import (
	"log"
	"testing"
)

func TestPutDeleteAndPrint(t *testing.T) {
	tree := NewBPlusTree(3)
	data := [][2]string{
		{"1", "a"},
		{"2", "c"},
		{"3", "b"},
		{"4", "d"},
		{"5", "e"},
		{"6", "fs"},
	}
	for _, kv := range data {
		err := tree.Put(kv[0], kv[1])
		tree.Print()
		log.Println("---")
		if err != nil {
			t.Fatalf("Put error: %v", err)
		}
	}

	err := tree.Put("7", "f")
	if err != nil {
		t.Fatalf("Put error: %v", err)
	}
	tree.Print()
	err = tree.Put("8", "f")
	if err != nil {
		t.Fatalf("Put error: %v", err)
	}
	tree.Print()
	err = tree.Put("9", "f")
	if err != nil {
		t.Fatalf("Put error: %v", err)
	}
	tree.Print()
	err = tree.Put("a", "f")
	if err != nil {
		t.Fatalf("Put error: %v", err)
	}
	tree.Print()
	err = tree.Put("b", "f")
	if err != nil {
		t.Fatalf("Put error: %v", err)
	}
	tree.Print()

	ok, err := tree.Del("1")
	if !ok {
		if err != nil {
			t.Fatalf("Put error: %v", err)
		}
	}
	tree.Print()

	ok, err = tree.Del("3")
	if !ok {
		if err != nil {
			t.Fatalf("Put error: %v", err)
		}
	}
	tree.Print()

	ok, err = tree.Del("6")
	if !ok {
		if err != nil {
			t.Fatalf("Put error: %v", err)
		}
	}
	tree.Print()

	ok, err = tree.Del("5")
	if !ok {
		if err != nil {
			t.Fatalf("Put error: %v", err)
		}
	}
	tree.Print()
	ok, err = tree.Del("4")
	if !ok {
		if err != nil {
			t.Fatalf("Put error: %v", err)
		}
	}
	tree.Print()

	ok, err = tree.Del("8")
	if !ok {
		if err != nil {
			t.Fatalf("Put error: %v", err)
		}
	}
	tree.Print()
}
