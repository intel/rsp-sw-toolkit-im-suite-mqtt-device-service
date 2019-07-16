package driver

import (
	"fmt"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"reflect"
	"strings"
	"testing"
)

func init() {
	driver = new(Driver)
	driver.Logger = logger.NewClient("test", false, "", "DEBUG")
}

func TestInvalidTopicMappingsSize(t *testing.T) {
	tests := []struct {
		topics      []string
		mappings    []string
		expectError bool
	}{
		{
			topics:      []string{"topic1", "topic2"},
			mappings:    []string{},
			expectError: true,
		},
		{
			topics:      []string{"topic1", "topic2", "topic3"},
			mappings:    []string{"mapping1", "mapping2", "mapping3"},
			expectError: false,
		},
		{
			topics:      []string{"topic1"},
			mappings:    []string{"mapping1", "mapping2", "mapping3"},
			expectError: true,
		},
		{
			topics:      []string{},
			mappings:    []string{},
			expectError: false,
		},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("%dtopics_%dmappings", len(test.topics), len(test.mappings)), func(t *testing.T) {
			conf := configuration{
				IncomingTopics:                test.topics,
				IncomingTopicResourceMappings: test.mappings,
			}

			_, err := compileTopicMappings(conf)
			if err != nil && !test.expectError {
				t.Errorf("did not expect error, but got %v\n\ttest: %#v", err, test)
			} else if err == nil && test.expectError {
				t.Errorf("expected to get an error, but result was nil\n\ttest: %#v", test)
			}
		})
	}
}

func TestTopicMappingRegexes(t *testing.T) {
	tests := []struct {
		// a single topic to simulate subscribing to (MAY CONTAIN WILDCARDS)
		topic   string
		// a list of non-wildcard topics that SHOULD match the generated regex pattern
		match   []string
		// a list of non-wildcard topics that SHOULD NOT match the generated regex pattern
		noMatch []string
	}{
		{
			topic:   "simple/topic",
			match:   []string{"simple/topic"},
			noMatch: []string{"simple_topic", "simple", "topic"},
		},
		{
			topic:   "single/level/wildcard/+",
			match:   []string{"single/level/wildcard/match", "single/level/wildcard/pass"},
			noMatch: []string{"single/level/wildcard", "fail/wildcard", "single/level/wildcard/one/two", "single/level/wildcard/one/two/three"},
		},
		{
			topic:   "multi/level/wildcard/#",
			match:   []string{"multi/level/wildcard/one", "multi/level/wildcard/one/two", "multi/level/wildcard/one/two/three"},
			noMatch: []string{"multi/level/wildcard", "multi/fail/wildcard", "fail/multi/level/wildcard"},
		},
		{
			topic:   "one/+/three/+/five",
			match:   []string{"one/two/three/four/five", "one/foo/three/bar/five"},
			noMatch: []string{"one/three/five", "one/two/three/four/foobar/five", "one/two/three/four/five/six"},
		},
		{
			topic:   "#",
			match:   []string{"this/will/literally/match/any/topic", "wildcard", "some/topic"},
			noMatch: []string{},
		},
		{
			topic:   "+",
			match:   []string{"simple", "topic"},
			noMatch: []string{"simple/topic", "simple/but/extra/topic"},
		},
		{
			topic:   "$SYS/broker/uptime",
			match:   []string{"$SYS/broker/uptime"},
			noMatch: []string{"SYS/broker/uptime", "$SYS", "broker/uptime"},
		},
	}
	for _, test := range tests {
		t.Run(strings.ReplaceAll(test.topic, "/", "_"), func(t *testing.T) {
			conf := configuration{
				IncomingTopics:                []string{test.topic},
				IncomingTopicResourceMappings: []string{"example_mapping"},
			}

			mappings, err := compileTopicMappings(conf)
			if err != nil {
				t.Error(err)
			}
			topicMappings = mappings

			for _, match := range test.match {
				if _, err := mapTopicToValueDescriptor(match); err != nil {
					t.Errorf("string %s did not match regex %v generated for topic %s",
						match, reflect.ValueOf(mappings).MapKeys()[0], test.topic)
				}
			}

			for _, noMatch := range test.noMatch {
				if _, err := mapTopicToValueDescriptor(noMatch); err == nil {
					t.Errorf("expected string %s to not match regex %#v generated for topic %s, but it did match",
						noMatch, reflect.ValueOf(mappings).MapKeys()[0], test.topic)
				}
			}
		})
	}
}
