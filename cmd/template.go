package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var templateOutputFile string

// templateCmd repräsentiert das Hauptkommando template
var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Generate post templates with placeholders",
	Long:  `Create ready-to-fill social media post structures for product launches, feature updates, or thought leadership.`,
}

// templateLaunchCmd repräsentiert das Unterkommando template launch
var templateLaunchCmd = &cobra.Command{
	Use:   "launch",
	Short: "Generate a product launch template",
	Long:  `Generate a Markdown file structured for a product launch announcement.`,
	Run: func(cmd *cobra.Command, args []string) {
		runTemplateGen("launch", launchTemplate)
	},
}

// templateFeatureCmd repräsentiert das Unterkommando template feature
var templateFeatureCmd = &cobra.Command{
	Use:   "feature",
	Short: "Generate a feature update template",
	Long:  `Generate a Markdown file structured for a feature release announcement (Twitter thread).`,
	Run: func(cmd *cobra.Command, args []string) {
		runTemplateGen("feature", featureTemplate)
	},
}

// templateThoughtCmd repräsentiert das Unterkommando template thought
var templateThoughtCmd = &cobra.Command{
	Use:   "thought",
	Short: "Generate a thought leadership template",
	Long:  `Generate a Markdown file structured for a thought leadership post (LinkedIn style).`,
	Run: func(cmd *cobra.Command, args []string) {
		runTemplateGen("thought", thoughtTemplate)
	},
}

func runTemplateGen(templateType, content string) {
	filePath := templateOutputFile
	if filePath == "" {
		filePath = fmt.Sprintf("%s.md", templateType)
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		reportTemplateError(fmt.Errorf("failed to get absolute path: %w", err), 2)
		return
	}

	// Datei schreiben
	if !DryRunFlag {
		if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
			reportTemplateError(fmt.Errorf("failed to write template file: %w", err), 2)
			return
		}
	}

	reportTemplateSuccess(templateType, absPath)
}

type templateSuccessJSON struct {
	OK   bool   `json:"ok"`
	Type string `json:"type"`
	File string `json:"file"`
}

type templateErrorJSON struct {
	OK     bool     `json:"ok"`
	Code   int      `json:"code"`
	Errors []string `json:"errors"`
}

func reportTemplateSuccess(templateType, file string) {
	if FormatFlag == "json" {
		out := templateSuccessJSON{
			OK:   true,
			Type: templateType,
			File: file,
		}
		jsonBytes, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		prefix := ""
		if DryRunFlag {
			prefix = "[DRY RUN] "
		}
		fmt.Printf("%sSuccessfully generated %q template at:\n  %s\n", prefix, templateType, file)
	}
}

func reportTemplateError(err error, exitCode int) {
	if FormatFlag == "json" {
		out := templateErrorJSON{
			OK:     false,
			Code:   exitCode,
			Errors: []string{err.Error()},
		}
		jsonBytes, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(os.Stderr, string(jsonBytes))
	} else {
		fmt.Fprintf(os.Stderr, "Template Error: %v\n", err)
	}
	exitFunc(exitCode)
}

func init() {
	templateCmd.PersistentFlags().StringVarP(&templateOutputFile, "output", "o", "", "Output file path (defaults to <type>.md)")
	
	templateCmd.AddCommand(templateLaunchCmd)
	templateCmd.AddCommand(templateFeatureCmd)
	templateCmd.AddCommand(templateThoughtCmd)
	
	rootCmd.AddCommand(templateCmd)
}

const launchTemplate = `---
platform: all
type: single
title: "Product Launch: [Product Name]"
campaign: "launch-[year]"
tags:
  - product
  - launch
---
🚀 Exciting news! Today we are officially launching [Product Name]!

Here is what you need to know:
- 🌟 Feature 1: [Short description]
- ⚡ Feature 2: [Short description]
- 🛠 Feature 3: [Short description]

Check it out here: [Link to product]

We can't wait to hear your feedback! 👇
`

const featureTemplate = `---
platform: twitter
type: thread
title: "Feature Update: [Feature Name]"
campaign: "updates-[year]"
---
## Tweet 1

We just shipped a highly requested feature to [Product Name]: [Feature Name]! 📦

Here is a quick thread on how it works and how it helps you. 👇

## Tweet 2

1/ What is it?
[Describe the problem this feature solves, and how it works].

## Tweet 3

2/ Key benefits:
- 🔥 Benefit 1: [Short description]
- 📈 Benefit 2: [Short description]

## Reply

Try it out today: [Link to feature]
Let us know what you think! 💬
`

const thoughtTemplate = `---
platform: linkedin
type: single
title: "Thought Leadership: [Topic]"
tags:
  - engineering
  - coding
  - career
---
Here is a hard truth about [Topic]:

[Interesting hook paragraph detailing a common misconception].

After years of working in this field, I've realized three key lessons:

1. [Lesson 1]
[Context or example].

2. [Lesson 2]
[Context or example].

3. [Lesson 3]
[Context or example].

What is your experience with [Topic]? Let's discuss in the comments! 👇
`
