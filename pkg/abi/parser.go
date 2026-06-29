package abi

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// Regex patterns for parsing annotations
	functionPattern = regexp.MustCompile(`^///\s*@xrpl-function\s+(\w+)`)
	paramPattern    = regexp.MustCompile(`^///\s*@param\s+(\w+)\s+(\w+)(?:\s+-\s+(.+))?`)
	returnPattern   = regexp.MustCompile(`^///\s*@return\s+(\w+)(?:\s+-\s+(.+))?`)
	flagPattern     = regexp.MustCompile(`^///\s*@flag\s+(\d+)`)

	// Pattern to match Rust function declaration
	rustFnPattern = regexp.MustCompile(`^\s*(?:pub\s+)?fn\s+(\w+)\s*\(`)
)

// Parser handles parsing Rust source files for ABI annotations
type Parser struct {
	sourceDir string
}

// NewParser creates a new ABI parser
func NewParser(sourceDir string) *Parser {
	return &Parser{sourceDir: sourceDir}
}

// ParseContract parses all Rust files in the contract directory
func (p *Parser) ParseContract(contractName string) (*ABI, error) {
	abi := &ABI{
		ContractName: contractName,
		Functions:    []Function{},
	}

	// Find all .rs files
	err := filepath.Walk(p.sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".rs") {
			functions, err := p.parseFile(path)
			if err != nil {
				return fmt.Errorf("failed to parse %s: %w", path, err)
			}
			abi.Functions = append(abi.Functions, functions...)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return abi, nil
}

// parseFile parses a single Rust file for function annotations
func (p *Parser) parseFile(filePath string) ([]Function, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var functions []Function
	scanner := bufio.NewScanner(file)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Check if this is an @xrpl-function annotation
		if match := functionPattern.FindStringSubmatch(line); match != nil {
			functionName := match[1]

			// Parse the function and its annotations
			fn, err := p.parseFunction(scanner, &lineNum, functionName, filePath)
			if err != nil {
				return nil, fmt.Errorf("%s:%d: %w", filePath, lineNum, err)
			}

			functions = append(functions, fn)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return functions, nil
}

// parseFunction parses a single function's annotations
func (p *Parser) parseFunction(scanner *bufio.Scanner, lineNum *int, functionName string, filePath string) (Function, error) {
	fn := Function{
		Name:       functionName,
		Parameters: []Parameter{},
	}

	currentFlag := 0 // Default flag value

	// Continue reading lines until we hit the actual function declaration
	for scanner.Scan() {
		*lineNum++
		line := scanner.Text()

		// Check for @param annotation
		if match := paramPattern.FindStringSubmatch(line); match != nil {
			paramName := match[1]
			paramType := match[2]
			description := ""
			if len(match) > 3 {
				description = match[3]
			}

			// Validate type
			if !IsValidType(paramType) {
				return fn, fmt.Errorf("invalid type '%s' for parameter '%s'. Valid types: %s",
					paramType, paramName, getValidTypesString())
			}

			param := Parameter{
				Name:        paramName,
				Type:        paramType,
				Flag:        currentFlag,
				Description: description,
			}

			fn.Parameters = append(fn.Parameters, param)
			continue
		}

		// Check for @flag annotation
		if match := flagPattern.FindStringSubmatch(line); match != nil {
			_, _ = fmt.Sscanf(match[1], "%d", &currentFlag)
			continue
		}

		// Check for @return annotation
		if match := returnPattern.FindStringSubmatch(line); match != nil {
			returnType := match[1]
			description := ""
			if len(match) > 2 {
				description = match[2]
			}

			// Validate return type
			if !IsValidType(returnType) {
				return fn, fmt.Errorf("invalid return type '%s'. Valid types: %s",
					returnType, getValidTypesString())
			}

			fn.Returns = &ReturnType{
				Type:        returnType,
				Description: description,
			}
			continue
		}

		// Check if we've reached the actual function declaration
		if match := rustFnPattern.FindStringSubmatch(line); match != nil {
			rustFnName := match[1]

			// Verify the function name matches
			if rustFnName != functionName {
				return fn, fmt.Errorf("function name mismatch: @xrpl-function says '%s' but Rust function is '%s'",
					functionName, rustFnName)
			}

			// Function parsing complete
			break
		}

		// If it's not a comment and not a function declaration, skip
		if !strings.HasPrefix(strings.TrimSpace(line), "///") &&
			!strings.HasPrefix(strings.TrimSpace(line), "//") &&
			strings.TrimSpace(line) != "" {
			break
		}
	}

	return fn, nil
}

// getValidTypesString returns a comma-separated string of valid types
func getValidTypesString() string {
	types := make([]string, 0, len(ValidXRPLTypes))
	for typeName := range ValidXRPLTypes {
		types = append(types, typeName)
	}
	return strings.Join(types, ", ")
}
