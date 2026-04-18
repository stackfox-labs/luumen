package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/Wh1teSlash/luau-parser/ast"
	"github.com/Wh1teSlash/luau-parser/lexer"
	"github.com/Wh1teSlash/luau-parser/parser"
)

const FileName = "project.config.luau"

var ErrConfigNotFound = errors.New("project.config.luau not found")

type Config struct {
	Project  ProjectConfig
	Install  InstallConfig
	Tools    map[string]string
	Packages map[string]string
	Tasks    map[string]TaskValue
}

type ProjectConfig struct {
	Name        string
	Version     string
	Author      string
	Description string
}

type InstallConfig struct {
	Tools    bool
	Packages bool
}

type TaskValue struct {
	Steps []string
}

func NewTaskValue(steps ...string) TaskValue {
	copied := append([]string(nil), steps...)
	return TaskValue{Steps: copied}
}

func (v TaskValue) AsRawValue() any {
	switch len(v.Steps) {
	case 0:
		return ""
	case 1:
		return v.Steps[0]
	default:
		return append([]string(nil), v.Steps...)
	}
}

type dataValue interface{}

type dataObject map[string]dataValue
type dataArray []dataValue

func Load(path string) (*Config, error) {
	if path == "" {
		return nil, errors.New("config path is required")
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrConfigNotFound, path)
		}
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	cfg, err := decode(contents)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %w", filepath.Base(path), err)
	}

	return cfg, nil
}

func LoadFromDir(dir string) (*Config, error) {
	return Load(filepath.Join(dir, FileName))
}

func Write(path string, cfg *Config) error {
	if path == "" {
		return errors.New("config path is required")
	}
	if cfg == nil {
		return errors.New("config is nil")
	}

	output, err := encode(cfg)
	if err != nil {
		return fmt.Errorf("failed to encode %s: %w", filepath.Base(path), err)
	}

	if err := os.WriteFile(path, []byte(output), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}

	return nil
}

func WriteToDir(dir string, cfg *Config) error {
	return Write(filepath.Join(dir, FileName), cfg)
}

func decode(contents []byte) (*Config, error) {
	root, err := parseRootObject(string(contents))
	if err != nil {
		return nil, err
	}
	return fromDataObject(root)
}

func parseRootObject(source string) (dataObject, error) {
	factory := ast.NewFactory()
	defer factory.Reset()

	p := parser.New(lexer.New(source), factory)
	program := p.ParseProgram()
	if parseErrors := p.Errors(); len(parseErrors) > 0 {
		return nil, fmt.Errorf("invalid Luau syntax: %s", joinParseErrors(parseErrors))
	}

	meaningful := make([]ast.Stmt, 0, len(program.Body))
	for _, stmt := range program.Body {
		switch stmt.(type) {
		case *ast.Comment, *ast.EmptyStatement:
			continue
		default:
			meaningful = append(meaningful, stmt)
		}
	}

	if len(meaningful) != 1 {
		return nil, errors.New("config must contain exactly one top-level return statement")
	}

	ret, ok := meaningful[0].(*ast.ReturnStatement)
	if !ok {
		return nil, errors.New("config must contain a top-level return statement")
	}
	if len(ret.Values) != 1 {
		return nil, configError(ret, "return statement must return exactly one table")
	}

	value, err := decodeExpr(ret.Values[0])
	if err != nil {
		return nil, err
	}

	root, ok := value.(dataObject)
	if !ok {
		return nil, configError(ret.Values[0], "return statement must return a table")
	}

	return root, nil
}

func joinParseErrors(errs []error) string {
	if len(errs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(errs))
	for _, err := range errs {
		parts = append(parts, err.Error())
	}
	return strings.Join(parts, "; ")
}

func decodeExpr(expr ast.Expr) (dataValue, error) {
	expr = unwrapParenExpr(expr)

	switch node := expr.(type) {
	case *ast.Literal:
		switch node.Type {
		case "string":
			value, ok := node.Value.(string)
			if !ok {
				return nil, configError(node, "string literal has unexpected value")
			}
			return value, nil
		case "boolean":
			value, ok := node.Value.(bool)
			if !ok {
				return nil, configError(node, "boolean literal has unexpected value")
			}
			return value, nil
		case "number":
			switch value := node.Value.(type) {
			case int64:
				return value, nil
			case float64:
				return value, nil
			default:
				return nil, configError(node, "number literal has unexpected value")
			}
		case "nil":
			return nil, configError(node, "nil values are not supported in config")
		default:
			return nil, configError(node, fmt.Sprintf("unsupported literal type %q", node.Type))
		}
	case *ast.TableLiteral:
		return decodeTableLiteral(node)
	case *ast.Identifier:
		return nil, configError(node, "identifier references are not allowed in config")
	case *ast.BinaryOp, *ast.UnaryOp, *ast.TypeCast, *ast.FieldAccess, *ast.IndexAccess, *ast.IfExpr, *ast.InterpolatedString:
		return nil, configError(expr, "computed expressions are not allowed in config")
	case *ast.FunctionCall, *ast.MethodCall:
		return nil, configError(expr, "function calls are not allowed in config")
	case *ast.FunctionExpr:
		return nil, configError(expr, "functions are not allowed in config")
	case *ast.VarArgs:
		return nil, configError(expr, "varargs are not allowed in config")
	default:
		return nil, configError(expr, "unsupported expression in config")
	}
}

func decodeTableLiteral(node *ast.TableLiteral) (dataValue, error) {
	if len(node.Fields) == 0 {
		return dataObject{}, nil
	}

	hasNamed := false
	hasArray := false
	for _, field := range node.Fields {
		if field.Key == nil {
			hasArray = true
			continue
		}
		hasNamed = true
	}

	if hasNamed && hasArray {
		return nil, configError(node, "mixed keyed and array table fields are not supported")
	}

	if hasArray {
		values := make(dataArray, 0, len(node.Fields))
		for _, field := range node.Fields {
			value, err := decodeExpr(field.Value)
			if err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		return values, nil
	}

	values := make(dataObject, len(node.Fields))
	for _, field := range node.Fields {
		key, err := decodeTableKey(field.Key)
		if err != nil {
			return nil, err
		}
		if _, exists := values[key]; exists {
			return nil, configError(field.Key, fmt.Sprintf("duplicate key %q", key))
		}

		value, err := decodeExpr(field.Value)
		if err != nil {
			return nil, err
		}
		values[key] = value
	}

	return values, nil
}

func decodeTableKey(expr ast.Expr) (string, error) {
	expr = unwrapParenExpr(expr)

	switch node := expr.(type) {
	case *ast.Literal:
		if node.Type != "string" {
			return "", configError(node, "table keys must be strings or identifiers")
		}
		value, ok := node.Value.(string)
		if !ok {
			return "", configError(node, "string table key has unexpected value")
		}
		if strings.TrimSpace(value) == "" {
			return "", configError(node, "table keys must not be empty")
		}
		return value, nil
	case *ast.Identifier:
		return "", configError(node, "identifier references are not allowed as table keys")
	default:
		return "", configError(expr, "table keys must be strings or identifiers")
	}
}

func unwrapParenExpr(expr ast.Expr) ast.Expr {
	for {
		paren, ok := expr.(*ast.ParenExpr)
		if !ok {
			return expr
		}
		expr = paren.Expr
	}
}

func fromDataObject(root dataObject) (*Config, error) {
	cfg := &Config{}

	allowedSections := map[string]struct{}{
		"project":  {},
		"install":  {},
		"tools":    {},
		"packages": {},
		"tasks":    {},
	}

	for _, key := range sortedObjectKeys(root) {
		if _, ok := allowedSections[key]; !ok {
			return nil, fmt.Errorf("unknown top-level section %q", key)
		}
	}

	project, err := parseProjectSection(root["project"])
	if err != nil {
		return nil, err
	}
	cfg.Project = project

	install, err := parseInstallSection(root["install"])
	if err != nil {
		return nil, err
	}
	cfg.Install = install

	tools, err := parseStringMapSection("tools", root["tools"])
	if err != nil {
		return nil, err
	}
	cfg.Tools = tools

	packages, err := parseStringMapSection("packages", root["packages"])
	if err != nil {
		return nil, err
	}
	cfg.Packages = packages

	tasks, err := parseTaskMapSection("tasks", root["tasks"])
	if err != nil {
		return nil, err
	}
	cfg.Tasks = tasks

	return cfg, nil
}

func parseProjectSection(value dataValue) (ProjectConfig, error) {
	if value == nil {
		return ProjectConfig{}, nil
	}

	object, ok := value.(dataObject)
	if !ok {
		return ProjectConfig{}, errors.New("project must be a table")
	}

	cfg := ProjectConfig{}
	allowedKeys := map[string]struct{}{
		"name":        {},
		"version":     {},
		"author":      {},
		"description": {},
	}

	for _, key := range sortedObjectKeys(object) {
		if _, ok := allowedKeys[key]; !ok {
			return ProjectConfig{}, fmt.Errorf("project.%s is not supported", key)
		}

		stringValue, err := requireString("project."+key, object[key])
		if err != nil {
			return ProjectConfig{}, err
		}

		switch key {
		case "name":
			cfg.Name = stringValue
		case "version":
			cfg.Version = stringValue
		case "author":
			cfg.Author = stringValue
		case "description":
			cfg.Description = stringValue
		}
	}

	return cfg, nil
}

func parseInstallSection(value dataValue) (InstallConfig, error) {
	if value == nil {
		return InstallConfig{}, nil
	}

	object, ok := value.(dataObject)
	if !ok {
		return InstallConfig{}, errors.New("install must be a table")
	}

	cfg := InstallConfig{}
	allowedKeys := map[string]struct{}{
		"tools":    {},
		"packages": {},
	}

	for _, key := range sortedObjectKeys(object) {
		if _, ok := allowedKeys[key]; !ok {
			return InstallConfig{}, fmt.Errorf("install.%s is not supported", key)
		}

		booleanValue, err := requireBool("install."+key, object[key])
		if err != nil {
			return InstallConfig{}, err
		}

		switch key {
		case "tools":
			cfg.Tools = booleanValue
		case "packages":
			cfg.Packages = booleanValue
		}
	}

	return cfg, nil
}

func parseStringMapSection(scope string, value dataValue) (map[string]string, error) {
	if value == nil {
		return nil, nil
	}

	object, ok := value.(dataObject)
	if !ok {
		return nil, fmt.Errorf("%s must be a table", scope)
	}
	if len(object) == 0 {
		return nil, nil
	}

	normalized := make(map[string]string, len(object))
	for _, key := range sortedObjectKeys(object) {
		stringValue, err := requireString(scope+"."+key, object[key])
		if err != nil {
			return nil, err
		}
		normalized[key] = stringValue
	}

	return normalized, nil
}

func parseTaskMapSection(scope string, value dataValue) (map[string]TaskValue, error) {
	if value == nil {
		return nil, nil
	}

	object, ok := value.(dataObject)
	if !ok {
		return nil, fmt.Errorf("%s must be a table", scope)
	}
	if len(object) == 0 {
		return nil, nil
	}

	normalized := make(map[string]TaskValue, len(object))
	for _, key := range sortedObjectKeys(object) {
		task, err := parseTaskValue(scope+"."+key, object[key])
		if err != nil {
			return nil, err
		}
		normalized[key] = task
	}

	return normalized, nil
}

func parseTaskValue(path string, value dataValue) (TaskValue, error) {
	switch typed := value.(type) {
	case string:
		steps, err := normalizeTaskSteps([]string{typed})
		if err != nil {
			return TaskValue{}, fmt.Errorf("%s: %w", path, err)
		}
		return TaskValue{Steps: steps}, nil
	case dataArray:
		steps := make([]string, 0, len(typed))
		for index, item := range typed {
			step, ok := item.(string)
			if !ok {
				return TaskValue{}, fmt.Errorf("%s[%d]: expected a string", path, index)
			}
			steps = append(steps, step)
		}
		normalized, err := normalizeTaskSteps(steps)
		if err != nil {
			return TaskValue{}, fmt.Errorf("%s: %w", path, err)
		}
		return TaskValue{Steps: normalized}, nil
	default:
		return TaskValue{}, fmt.Errorf("%s: expected a string or array of strings", path)
	}
}

func requireString(path string, value dataValue) (string, error) {
	stringValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s: expected a string", path)
	}
	return stringValue, nil
}

func requireBool(path string, value dataValue) (bool, error) {
	booleanValue, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("%s: expected a boolean", path)
	}
	return booleanValue, nil
}

func normalizeTaskSteps(steps []string) ([]string, error) {
	if len(steps) == 0 {
		return nil, errors.New("must contain at least one step")
	}

	normalized := make([]string, 0, len(steps))
	for _, step := range steps {
		trimmed := strings.TrimSpace(step)
		if trimmed == "" {
			return nil, errors.New("step must not be empty")
		}
		normalized = append(normalized, trimmed)
	}

	return normalized, nil
}

func sortedObjectKeys(values dataObject) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func configError(node ast.Node, message string) error {
	pos := node.Pos()
	if pos.Line == 0 && pos.Column == 0 {
		return errors.New(message)
	}
	return fmt.Errorf("%s at line %d, col %d", message, pos.Line, pos.Column)
}

type objectEntry struct {
	key   string
	value any
}

func encode(cfg *Config) (string, error) {
	topLevel := make([]objectEntry, 0, 5)

	if project := encodeProject(cfg.Project); len(project) > 0 {
		topLevel = append(topLevel, objectEntry{key: "project", value: project})
	}
	if install := encodeInstall(cfg.Install); len(install) > 0 {
		topLevel = append(topLevel, objectEntry{key: "install", value: install})
	}
	if tools := encodeStringMap(cfg.Tools); len(tools) > 0 {
		topLevel = append(topLevel, objectEntry{key: "tools", value: tools})
	}
	if packages := encodeStringMap(cfg.Packages); len(packages) > 0 {
		topLevel = append(topLevel, objectEntry{key: "packages", value: packages})
	}
	if tasks, err := encodeTaskMap("tasks", cfg.Tasks); err != nil {
		return "", err
	} else if len(tasks) > 0 {
		topLevel = append(topLevel, objectEntry{key: "tasks", value: tasks})
	}

	var builder strings.Builder
	builder.WriteString("return ")
	writeObject(&builder, topLevel, 0)
	builder.WriteString("\n")
	return builder.String(), nil
}

func encodeProject(project ProjectConfig) []objectEntry {
	entries := make([]objectEntry, 0, 4)
	if strings.TrimSpace(project.Name) != "" {
		entries = append(entries, objectEntry{key: "name", value: project.Name})
	}
	if strings.TrimSpace(project.Version) != "" {
		entries = append(entries, objectEntry{key: "version", value: project.Version})
	}
	if strings.TrimSpace(project.Author) != "" {
		entries = append(entries, objectEntry{key: "author", value: project.Author})
	}
	if strings.TrimSpace(project.Description) != "" {
		entries = append(entries, objectEntry{key: "description", value: project.Description})
	}
	return entries
}

func encodeInstall(install InstallConfig) []objectEntry {
	entries := make([]objectEntry, 0, 2)
	if install.Tools {
		entries = append(entries, objectEntry{key: "tools", value: true})
	}
	if install.Packages {
		entries = append(entries, objectEntry{key: "packages", value: true})
	}
	return entries
}

func encodeStringMap(values map[string]string) []objectEntry {
	if len(values) == 0 {
		return nil
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	entries := make([]objectEntry, 0, len(keys))
	for _, key := range keys {
		entries = append(entries, objectEntry{key: key, value: values[key]})
	}
	return entries
}

func encodeTaskMap(scope string, values map[string]TaskValue) ([]objectEntry, error) {
	if len(values) == 0 {
		return nil, nil
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	entries := make([]objectEntry, 0, len(keys))
	for _, key := range keys {
		steps, err := normalizeTaskSteps(values[key].Steps)
		if err != nil {
			return nil, fmt.Errorf("%s.%s: %w", scope, key, err)
		}
		if len(steps) == 1 {
			entries = append(entries, objectEntry{key: key, value: steps[0]})
			continue
		}
		entries = append(entries, objectEntry{key: key, value: steps})
	}

	return entries, nil
}

func writeObject(builder *strings.Builder, entries []objectEntry, indent int) {
	if len(entries) == 0 {
		builder.WriteString("{}")
		return
	}

	builder.WriteString("{\n")
	for index, entry := range entries {
		builder.WriteString(strings.Repeat("    ", indent+1))
		writeKey(builder, entry.key)
		builder.WriteString(" = ")
		writeValue(builder, entry.value, indent+1)
		builder.WriteString(",\n")
		if index == len(entries)-1 {
			continue
		}
	}
	builder.WriteString(strings.Repeat("    ", indent))
	builder.WriteString("}")
}

func writeValue(builder *strings.Builder, value any, indent int) {
	switch typed := value.(type) {
	case string:
		builder.WriteString(strconv.Quote(typed))
	case bool:
		if typed {
			builder.WriteString("true")
		} else {
			builder.WriteString("false")
		}
	case int:
		builder.WriteString(strconv.Itoa(typed))
	case int64:
		builder.WriteString(strconv.FormatInt(typed, 10))
	case float64:
		builder.WriteString(strconv.FormatFloat(typed, 'f', -1, 64))
	case []string:
		writeStringArray(builder, typed, indent)
	case []objectEntry:
		writeObject(builder, typed, indent)
	default:
		builder.WriteString("nil")
	}
}

func writeStringArray(builder *strings.Builder, values []string, indent int) {
	if len(values) == 0 {
		builder.WriteString("{}")
		return
	}

	builder.WriteString("{\n")
	for _, value := range values {
		builder.WriteString(strings.Repeat("    ", indent+1))
		builder.WriteString(strconv.Quote(value))
		builder.WriteString(",\n")
	}
	builder.WriteString(strings.Repeat("    ", indent))
	builder.WriteString("}")
}

func writeKey(builder *strings.Builder, key string) {
	if isIdentifierKey(key) {
		builder.WriteString(key)
		return
	}
	builder.WriteString("[")
	builder.WriteString(strconv.Quote(key))
	builder.WriteString("]")
}

func isIdentifierKey(key string) bool {
	if key == "" || luauKeywords[key] {
		return false
	}

	for index, r := range key {
		if index == 0 {
			if r != '_' && !unicode.IsLetter(r) {
				return false
			}
			continue
		}

		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}

	return true
}

var luauKeywords = map[string]bool{
	"and": true, "break": true, "continue": true, "do": true, "else": true, "elseif": true,
	"end": true, "export": true, "false": true, "for": true, "function": true, "if": true,
	"in": true, "local": true, "nil": true, "not": true, "or": true, "repeat": true,
	"return": true, "then": true, "true": true, "type": true, "until": true, "while": true,
}
