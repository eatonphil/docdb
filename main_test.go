package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_getPath(t *testing.T) {
	tests := []struct {
		object        map[string]any
		path          []string
		expectedValue any
		expectedOk    bool
	}{
		{
			map[string]any{
				"a": map[string]any{
					"b": 1,
				},
			},
			[]string{"a", "b"},
			1,
			true,
		},
		{
			map[string]any{
				"a": map[string]any{
					"b": 1,
				},
			},
			[]string{"a", "c"},
			nil,
			false,
		},
	}

	for _, test := range tests {
		v, ok := getPath(test.object, test.path)
		assert.Equal(t, test.expectedValue, v)
		assert.Equal(t, test.expectedOk, ok)
	}
}

func Test_lexString(t *testing.T) {
	tests := []struct {
		input          string
		index          int
		expectedString string
		expectedIndex  int
		expectedErr    error
	}{
		{
			"a.b:c",
			0,
			"a.b",
			3,
			nil,
		},
		{
			`"a b : . 2":12`,
			0,
			"a b : . 2",
			11,
			nil,
		},
		{
			` a:2`,
			0,
			"",
			0,
			fmt.Errorf("No string found"),
		},
		{
			` a:2`,
			1,
			"a",
			2,
			nil,
		},
	}

	for _, test := range tests {
		s, outIndex, err := lexString([]rune(test.input), test.index)
		assert.Equal(t, test.expectedString, s)
		assert.Equal(t, test.expectedIndex, outIndex)
		assert.Equal(t, test.expectedErr, err)
	}
}

func Test_parseQuery(t *testing.T) {
	tests := []struct {
		q             string
		expectedQuery query
		expectedErr   error
	}{
		{
			"a.b:1 c:2",
			query{
				[]queryEquals{
					{
						key:   []string{"a", "b"},
						value: "1",
					},
					{
						key:   []string{"c"},
						value: "2",
					},
				},
			},
			nil,
		},
		{
			"a:1",
			query{
				[]queryEquals{
					{
						key:   []string{"a"},
						value: "1",
					},
				},
			},
			nil,
		},
		{
			`" a ":" n "`,
			query{
				[]queryEquals{
					{
						key:   []string{" a "},
						value: " n ",
					},
				},
			},
			nil,
		},
		{
			"",
			query{},
			nil,
		},
	}

	for _, test := range tests {
		query, err := parseQuery(test.q)
		if query == nil {
			fmt.Println(test, err)
		}
		assert.Equal(t, test.expectedQuery, *query)
		assert.Equal(t, test.expectedErr, err)
	}
}
