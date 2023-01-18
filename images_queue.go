package main

type ProcessQueue []*Image

// Push required by heap.Interface
func (iq *ProcessQueue) Push(imageData interface{}) {
	image := imageData.(*Image)
	image.index = len(*iq) // required by the heap.Interface
	*iq = append(*iq, image)
}

// Pop required by heap.Interface
func (iq *ProcessQueue) Pop() interface{} {
	currentQueue := *iq
	n := len(currentQueue)

	image := currentQueue[n-1]
	image.index = -1
	*iq = currentQueue[0 : n-1]
	return image
}

func (iq *ProcessQueue) Terminate(image *Image, workerID int) {
	currentQueue := *iq
	currentQueue = append(currentQueue[:image.index], currentQueue[image.index+1:]...)
	image.index = -1
	*iq = currentQueue
}

func (iq *ProcessQueue) Peek() interface{} {
	currentQueue := *iq
	return currentQueue[0]
}

func (iq *ProcessQueue) PeekForObject() *Image {
	currentQueue := *iq
	return currentQueue[0]
}

// Len required by sort.Interface
func (iq *ProcessQueue) Len() int {
	return len(*iq)
}

func (iq ProcessQueue) Swap(a, b int) {
	iq[a], iq[b] = iq[b], iq[a]
	iq[a].index = a
	iq[b].index = b
}

func (iq ProcessQueue) Less(a, b int) bool {
	return iq[a].priority < iq[b].priority
}
