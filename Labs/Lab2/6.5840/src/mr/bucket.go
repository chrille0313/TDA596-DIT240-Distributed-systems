package mr

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func readBucketFile(taskID TaskID, bucket int) ([]KeyValue, error) {
	filename := taskIntermediateFileName(taskID, bucket)
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("worker: cannot open %s: %w", filename, err)
	}
	defer file.Close()

	keyValues := make([]KeyValue, 0)
	decoder := json.NewDecoder(file)
	for {
		var kv KeyValue
		err := decoder.Decode(&kv)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("worker: cannot decode %s: %w", filename, err)
		}
		keyValues = append(keyValues, kv)
	}
	return keyValues, nil
}

func writeReducedContentsToBucketFile(bucket int, reducedContents []KeyValue) error {
	filename := reduceOutputFileName(bucket)
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("worker: cannot create %s: %w", filename, err)
	}
	defer file.Close()

	for _, kv := range reducedContents {
		fmt.Fprintf(file, "%v %v\n", kv.Key, kv.Value)
	}

	return nil
}

func taskIntermediateFileName(taskID TaskID, bucket int) string {
	return fmt.Sprintf("mr-%d-%d", taskID, bucket)
}
