package util

import (
	"sort"

	"github.com/fatih/color"
)

var ColorInfo = color.New(color.FgGreen).SprintFunc()
var ColorStatus = color.New(color.FgBlue).SprintFunc()
var ColorWarning = color.New(color.FgYellow).SprintFunc()
var ColorError = color.New(color.FgRed).SprintFunc()

var colorMap = map[string]color.Attribute{
	// formatting
	"bold":         color.Bold,
	"faint":        color.Faint,
	"italic":       color.Italic,
	"underline":    color.Underline,
	"blinkslow":    color.BlinkSlow,
	"blinkrapid":   color.BlinkRapid,
	"reversevideo": color.ReverseVideo,
	"concealed":    color.Concealed,
	"crossedout":   color.CrossedOut,

	// Foreground text colors
	"black":   color.FgBlack,
	"red":     color.FgRed,
	"green":   color.FgGreen,
	"yellow":  color.FgYellow,
	"blue":    color.FgBlue,
	"magenta": color.FgMagenta,
	"cyan":    color.FgCyan,
	"white":   color.FgWhite,

	// Foreground Hi-Intensity text colors
	"hiblack":   color.FgHiBlack,
	"hired":     color.FgHiRed,
	"higreen":   color.FgHiGreen,
	"hiyellow":  color.FgHiYellow,
	"hiblue":    color.FgHiBlue,
	"himagenta": color.FgHiMagenta,
	"hicyan":    color.FgHiCyan,
	"hiwhite":   color.FgHiWhite,

	// Background text colors
	"bgblack":   color.BgBlack,
	"bgred":     color.BgRed,
	"bggreen":   color.BgGreen,
	"bgyellow":  color.BgYellow,
	"BgBlue":    color.BgBlue,
	"bgmagenta": color.BgMagenta,
	"bgcyan":    color.BgCyan,
	"bgwhite":   color.BgWhite,

	// Background Hi-Intensity text colors
	"bghiblack":   color.BgHiBlack,
	"bghired":     color.BgHiRed,
	"bghigreen":   color.BgHiGreen,
	"bghiyellow":  color.BgHiYellow,
	"bghiblue":    color.BgHiBlue,
	"bghimagenta": color.BgHiMagenta,
	"bghicyan":    color.BgHiCyan,
	"bghiwhite":   color.BgHiWhite,
}

// GetColor returns the color for the list of colour names and option name
func GetColor(optionName string, colorNames []string) (*color.Color, error) {
	attributes := []color.Attribute{}
	for _, colorName := range colorNames {
		a := colorMap[colorName]
		if a == color.Attribute(0) {
			return nil, InvalidOption(optionName, colorName, ColorNameValues())
		}
		attributes = append(attributes, a)
	}
	return color.New(attributes...), nil
}

// ColorNameValues returns all the color names sorted
func ColorNameValues() []string {
	answer := []string{}
	for k, _ := range colorMap {
		answer = append(answer, k)
	}
	sort.Strings(answer)
	return answer
}
