package variables

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/nubank/workflows/pkg/github"
	"k8s.io/client-go/util/jsonpath"
)

const (
	nothing = ""

	null = "null"
)

// query evaluates the provided JSON path expression against the supplied Event object and
// returns results.
func query(event *github.Event, expression string) (string, error) {
	jsonPath := jsonpath.New("workflow.event").AllowMissingKeys(true)
	if err := jsonPath.Parse(expression); err != nil {
		return nothing, err
	}

	results, err := jsonPath.FindResults(event.Data)
	if err != nil {
		return nothing, err
	}

	if len(results) > 1 {
		return nothing, errors.New("JSON path expressions with multiple results aren't supported")
	}

	innerResults := results[0]
	if len(innerResults) == 1 {
		return printElement(innerResults[0])
	}
	return printList(innerResults)
}

func printElement(runtimeValue reflect.Value) (string, error) {
	if runtimeValue.Kind() == reflect.Ptr {
		runtimeValue = runtimeValue.Elem()
	}

	if runtimeValue.Kind() == reflect.Invalid {
		return null, nil
	}

	value := runtimeValue.Interface()

	if runtimeValue.Kind() == reflect.String {
		return fmt.Sprint(value), nil
	}
	return printJSON(value)
}

func printList(results []reflect.Value) (string, error) {
	values := make([]interface{}, 0)
	for _, value := range results {
		var underwingValue interface{}
		if value.Kind() != reflect.Invalid {
			underwingValue = value.Interface()
		} else {
			underwingValue = null
		}

		values = append(values, underwingValue)
	}
	return printJSON(values)
}

func printJSON(value interface{}) (string, error) {
	jsonResult, err := json.Marshal(value)
	if err != nil {
		return nothing, err
	}
	return string(jsonResult), nil
}
