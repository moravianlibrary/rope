package main

type Priority int

type Image struct {
	Input    string
	Output   string
	WorkerID int
	priority int
	state    int
	lenght   int

	index int
}
