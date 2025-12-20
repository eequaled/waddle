package storage

import (
	"fmt"
	"reflect"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
)

// GenSession generates random Session instances for property testing.
func GenSession() gopter.Gen {
	return gen.Struct(reflect.TypeOf(Session{}), map[string]gopter.Gen{
		"ID":              gen.Int64Range(1, 1000000),
		"Date":            GenDateString(),
		"CustomTitle":     gen.AnyString(),
		"CustomSummary":   gen.AnyString(),
		"OriginalSummary": gen.AnyString(),
		"ExtractedText":   gen.AnyString(),
		"CreatedAt":       GenTime(),
		"UpdatedAt":       GenTime(),
	})
}

// GenActivityBlock generates random ActivityBlock instances for property testing.
func GenActivityBlock() gopter.Gen {
	return gen.Struct(reflect.TypeOf(ActivityBlock{}), map[string]gopter.Gen{
		"ID":            gen.Int64Range(1, 1000000),
		"AppActivityID": gen.Int64Range(1, 1000000),
		"BlockID":       GenBlockID(),
		"StartTime":     GenTime(),
		"EndTime":       GenTime(),
		"OCRText":       gen.AnyString(),
		"MicroSummary":  gen.AnyString(),
	})
}

// GenAppActivity generates random AppActivity instances for property testing.
func GenAppActivity() gopter.Gen {
	return gen.Struct(reflect.TypeOf(AppActivity{}), map[string]gopter.Gen{
		"ID":        gen.Int64Range(1, 1000000),
		"SessionID": gen.Int64Range(1, 1000000),
		"AppName":   GenAppName(),
		"CreatedAt": GenTime(),
		"UpdatedAt": GenTime(),
	})
}

// GenChatMessage generates random ChatMessage instances for property testing.
func GenChatMessage() gopter.Gen {
	return gen.Struct(reflect.TypeOf(ChatMessage{}), map[string]gopter.Gen{
		"ID":        gen.Int64Range(1, 1000000),
		"SessionID": gen.Int64Range(1, 1000000),
		"Role":      GenChatRole(),
		"Content":   gen.AnyString(),
		"Timestamp": GenTime(),
	})
}

// GenNotification generates random Notification instances for property testing.
func GenNotification() gopter.Gen {
	return gen.Struct(reflect.TypeOf(Notification{}), map[string]gopter.Gen{
		"ID":         gen.AlphaString(),
		"Type":       gen.OneConstOf("info", "warning", "error", "success"),
		"Title":      gen.AnyString(),
		"Message":    gen.AnyString(),
		"Timestamp":  GenTime(),
		"Read":       gen.Bool(),
		"SessionRef": gen.AnyString(),
		"Metadata":   gen.Const("{}"),
	})
}

// GenDateString generates valid date strings in "2006-01-02" format.
func GenDateString() gopter.Gen {
	return gopter.CombineGens(
		gen.IntRange(2020, 2030),
		gen.IntRange(1, 12),
		gen.IntRange(1, 28), // Safe for all months
	).Map(func(values []interface{}) string {
		year := values[0].(int)
		month := values[1].(int)
		day := values[2].(int)
		return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
	})
}

// GenBlockID generates valid block IDs in "HH-MM" format.
func GenBlockID() gopter.Gen {
	return gopter.CombineGens(
		gen.IntRange(0, 23),
		gen.IntRange(0, 59),
	).Map(func(values []interface{}) string {
		hour := values[0].(int)
		minute := values[1].(int)
		return fmt.Sprintf("%02d-%02d", hour, minute)
	})
}

// GenTime generates random time.Time values within a reasonable range.
func GenTime() gopter.Gen {
	return gen.Int64Range(
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		time.Date(2030, 12, 31, 23, 59, 59, 0, time.UTC).Unix(),
	).Map(func(unix int64) time.Time {
		return time.Unix(unix, 0).UTC()
	})
}

// GenChatRole generates valid chat roles.
func GenChatRole() gopter.Gen {
	return gen.OneConstOf(ChatRoleUser, ChatRoleAssistant)
}

// GenAppName generates realistic application names.
func GenAppName() gopter.Gen {
	return gen.OneConstOf(
		"Chrome",
		"Firefox",
		"Visual Studio Code",
		"Slack",
		"Discord",
		"Microsoft Word",
		"Excel",
		"PowerPoint",
		"Notepad",
		"Terminal",
		"Explorer",
		"Spotify",
		"Zoom",
		"Teams",
	)
}

// GenEmbedding generates random 768-dimensional embedding vectors.
func GenEmbedding() gopter.Gen {
	return gen.SliceOfN(768, gen.Float32Range(-1.0, 1.0))
}

// GenSearchQuery generates random search query strings.
func GenSearchQuery() gopter.Gen {
	return gen.AnyString().SuchThat(func(s string) bool {
		return len(s) > 0
	})
}

// GenDateRange generates valid DateRange instances.
func GenDateRange() gopter.Gen {
	return gopter.CombineGens(
		GenDateString(),
		GenDateString(),
	).Map(func(values []interface{}) DateRange {
		startDate := values[0].(string)
		endDate := values[1].(string)
		// Ensure start <= end
		if startDate > endDate {
			startDate, endDate = endDate, startDate
		}
		return DateRange{
			StartDate: startDate,
			EndDate:   endDate,
		}
	})
}

// GenNonEmptyString generates non-empty strings for required fields.
func GenNonEmptyString() gopter.Gen {
	return gen.AnyString().SuchThat(func(s string) bool {
		return len(s) > 0
	})
}

// GenUnicodeString generates strings with unicode characters for encryption testing.
func GenUnicodeString() gopter.Gen {
	return gen.AnyString()
}

// DefaultTestParameters returns gopter test parameters with minimum 100 iterations.
func DefaultTestParameters() *gopter.TestParameters {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	params.MaxSize = 1000
	return params
}
