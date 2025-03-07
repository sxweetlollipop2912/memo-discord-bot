package timeutil

import (
	"fmt"
	"time"

	"github.com/olebedev/when"
	"github.com/olebedev/when/rules/common"
	"github.com/olebedev/when/rules/en"
)

var w *when.Parser

func init() {
	w = when.New(nil)

	// Add English rules
	w.Add(en.All...)
	// Add common rules
	w.Add(common.All...)
}

// ParseTime converts natural language time expressions into time.Time
func ParseTime(input string) (time.Time, error) {
	// Parse the natural language time expression
	result, err := w.Parse(input, time.Now())
	if err != nil {
		return time.Time{}, fmt.Errorf("could not parse time. You can write in natural language, like:\n- in 2 hours\n- tomorrow at 3pm\n- next friday at 2pm")
	}
	if result == nil {
		return time.Time{}, fmt.Errorf("could not parse time. You can write in natural language, like:\n- in 2 hours\n- tomorrow at 3pm\n- next friday at 2pm")
	}

	return result.Time, nil
}
