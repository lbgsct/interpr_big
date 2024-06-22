package main

import (
	"fmt"
	"os"
	"regexp"
	"math"
	//"io"
	"bufio"
	"strconv"
	"strings"
)

//интерпретатор
type Interpreter struct {
	Commands       map[string]string
	Variables      *Trie
	BaseInput      int
	BaseOutput     int
	BaseAssign     int
	Result 		   string
	UnarySyntax    string
	BinarySyntax   string
	Debug          bool
	SettingsFile   string
	Oper           []string
}
 

//создание
func NewInterpreter(settingsFile string, baseInput, baseOutput, baseAssign int, debug bool) *Interpreter {
	interpreter := &Interpreter{
		Commands: map[string]string{
			"not":    "not",
			"input":  "input",
			"output": "output",
			"add":    "add",
			"mult":   "mult",
			"sub":    "sub",
			"pow":    "pow",
			"div":    "div",
			"rem":    "rem",
			"xor":    "xor",
			"and":    "and",
			"or":     "or",
			"=":      "=",
		},
		Variables:      NewTrie(),
		BaseInput:      baseInput,
		BaseOutput:     baseOutput,
		BaseAssign:     baseAssign,
		Result:         "left",
		UnarySyntax:    "op()",
		BinarySyntax:   "op()",
		Debug:          debug,
		SettingsFile:   settingsFile,
	}
	interpreter.LoadSettings(settingsFile)
	interpreter.SaveLastSettings()

	for original := range interpreter.Commands {
		interpreter.Oper = append(interpreter.Oper, original)
	}

	return interpreter
}


//сохранение в файл результата
func (interp *Interpreter) SaveLastSettings() error {
	file, err := os.Create("last_settings.txt")
	if err != nil {
		return err
	}
	defer file.Close()

	// Записываем данные в файл
	fmt.Fprintf(file, "settings_file=%s\n", interp.SettingsFile)

	return nil
}


//загрузка из файла 
func (interp *Interpreter) LoadSettings(settingsFile string) error {
	file, err := os.Open(settingsFile)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		switch line {
		case "left=":
			interp.Result = "left"
		case "right=":
			interp.Result = "right"
		case "op()", "()op":
			interp.BinarySyntax = line
			interp.UnarySyntax = line
		case "(op)":
			interp.BinarySyntax = line
		default:
			parts := strings.Fields(line)
			if len(parts) == 2 {
				interp.Commands[parts[0]] = parts[1]
			} else if len(parts) == 3 && strings.HasPrefix(parts[0], "[") {
				key := parts[1]
				value := strings.TrimRight(parts[2], "]")
				interp.Commands[key] = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

//разделение по командам
func (interp *Interpreter) Execute(program string) {
	lines := strings.Split(program, ";")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = interp.RemoveNestedComments(line)
		if line != "" {
			if interp.Debug && strings.Contains(line, "#BREAKPOINT") {
				interp.DebugPrompt()
				line = strings.ReplaceAll(line, "#BREAKPOINT", "")
			}

			line = interp.RemoveComments(line)
			interp.ProcessLine(line)
		}
	}
}

//удаление комментария
func (interp *Interpreter) RemoveComments(line string) string {
	commentIndex := strings.Index(line, "#")
	if commentIndex != -1 {
		line = line[:commentIndex]
	}
	return strings.TrimSpace(line)
}


//удаление блока комментарий
func (interp *Interpreter) RemoveNestedComments(text string) string {
	pattern := `\[[^\[\]]*?\]`
	re := regexp.MustCompile(pattern)
	for re.MatchString(text) {
		text = re.ReplaceAllString(text, "")
	}
	return text
}

//обработка команд
func (interp *Interpreter) ProcessLine(line string) {
	for original, synonym := range interp.Commands {
		str1 := synonym + "("
		str2 := ")" + synonym
		str3 := " " + synonym + " "
		if strings.Contains(line, str1) || strings.Contains(line, str2) || strings.Contains(line, str3) {
			line = strings.ReplaceAll(line, synonym, original)
		}
	}

	if strings.Contains(line, "=") {
		parts := strings.SplitN(line, "=", 2)
		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])

		if interp.Result == "left" {
			variable, expression := left, right
			if strings.Contains(line, "input()") {
				fmt.Printf("Enter value for %s: ", variable)
				var inputVal int
				fmt.Scanln(&inputVal)
				interp.Variables.Insert(variable, inputVal)
			} else {
				value := interp.EvaluateExpression(expression)
				interp.Variables.Insert(variable, value)
			}
		} else {
			expression, variable := left, right
			if strings.Contains(line, "input()") {
				fmt.Printf("Enter value for %s: ", variable)
				var inputVal int
				fmt.Scanln(&inputVal)
				interp.Variables.Insert(variable, inputVal)
			} else {
				value := interp.EvaluateExpression(expression)
				interp.Variables.Insert(variable, value)
			}
		}
	} else {
		expression := strings.TrimSpace(line)
		interp.EvaluateExpression(expression)
	}
}

//вычисляем выражение
func (interp *Interpreter) EvaluateExpression(expr string) int {
	if strings.Contains(expr, "output") {
		expr = strings.TrimSpace(expr)

		var varName string
		if strings.HasPrefix(expr, "output") {
			if interp.UnarySyntax != "op()" {
				panic("Ошибка: недопустимое расположение операндов и операций")
			}
			varName = strings.TrimSpace(expr[7 : len(expr)-1])
		} else if strings.HasSuffix(expr, "output") {
			if interp.UnarySyntax != "()op" {
				panic("Ошибка: недопустимое расположение операндов и операций")
			}
			varName = strings.TrimSpace(expr[1 : len(expr)-7])
		}

		if strings.Contains(varName, "output") || strings.Contains(varName, "input") {
			panic("ошибка")
		}

		value := interp.EvaluateExpression(varName)
		baseOutputValue := interp.DecimalToBase(value, interp.BaseOutput)
		fmt.Printf("%s = %s\n", varName, baseOutputValue)
		return value
	} else if matched, _ := regexp.MatchString(`^[0-9A-Fa-f]+$`, expr); matched {
		num, err := strconv.ParseInt(expr, 16, 64)
		if err != nil {
			panic(fmt.Sprintf("Ошибка при парсинге шестнадцатеричного числа: %v", err))
		}
		return int(num)
	} else {
		return int(interp.EvaluateInfix(expr))
	}
}

//10
func (interp *Interpreter) DecimalToBase(num, base int) string {
	if num == 0 {
		return "0"
	}

	digits := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	var result strings.Builder

	for num != 0 {
		remainder := num % base
		result.WriteByte(digits[remainder])
		num = num / base
	}

	// Переворачиваем строку, так как result записывает цифры в обратном порядке
	reversed := result.String()
	runes := []rune(reversed)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}

	return string(runes)
}


func (interp *Interpreter) EvaluateInfix(expression string) float64 {
	precedence := map[string]int{
		"add": 1,
		"sub": 1,
		"mult": 2,
		"div": 2,
		"rem": 2,
		"xor": 1,
		"and": 1,
		"or": 1,
		"pow": 3,
	}

	higherPrecedence := func(op1, op2 string) bool {
		return precedence[op1] >= precedence[op2]
	}

	var postfix []string
	var stack []string

	tokens := interp.Tokenize(expression)

	for _, token := range tokens {
		if interp.IsNumber(token) {
			postfix = append(postfix, token)
		} else if token == "(" {
			stack = append(stack, token)
		} else if token == ")" {
			for len(stack) > 0 && stack[len(stack)-1] != "(" {
				postfix = append(postfix, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			stack = stack[:len(stack)-1] // Pop "(" from stack
		} else if _, ok := precedence[token]; ok {
			for len(stack) > 0 && stack[len(stack)-1] != "(" && higherPrecedence(stack[len(stack)-1], token) {
				postfix = append(postfix, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			stack = append(stack, token)
		} else {
			panic("Invalid token: " + token)
		}
	}

	for len(stack) > 0 {
		postfix = append(postfix, stack[len(stack)-1])
		stack = stack[:len(stack)-1]
	}

	result, err := interp.EvalPostfix(postfix)
	if err != nil {
		panic("Error evaluating postfix expression: " + err.Error())
	}

	return result
}



//проверка на число
func (interp *Interpreter) IsNumber(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}



//токен
func (interp *Interpreter) Tokenize(expression string, evaluateInfix func(string) float64, executeCommand func(string, []string) string, variables map[string]interface{}) []string {
	expression = strings.TrimSpace(expression)
	var tokens []string
	var currentToken string
	var openB int
	i := 0
	for i != len(expression) {
		if expression[i] == ' ' && currentToken != "" && openB == 0 {
			if contains([]string{"add", "sub", "pow", "div", "rem", "xor", "and"}, currentToken) && binarySyntax != "(op)" {
				panic("Ошибка: недопустимое расположение операндов и операций")
			}
			tokens = append(tokens, currentToken)
			currentToken = ""
		} else if !strings.Contains("()", string(expression[i])) && expression[i] != ' ' {
			currentToken += string(expression[i])
		} else if expression[i] == '(' {
			currentToken += string(expression[i])
			openB++
		} else if expression[i] == ')' && openB != 0 {
			currentToken += string(expression[i])
			openB--
			if openB == 0 {
				var funcName string
				var args []string
				var current string
				var open int
				for j := 0; j < len(currentToken); j++ {
					if currentToken[j] == '(' {
						open++
						current += string(currentToken[j])
					} else if currentToken[j] == ')' {
						open--
						current += string(currentToken[j])
					} else if currentToken[j] == ',' && open == 0 {
						args = append(args, current)
						current = ""
					} else {
						current += string(currentToken[j])
					}
					if j == len(currentToken)-1 {
						args = append(args, current)
					}
				}

				if funcName == "" && len(args) == 1 {
					st := expression[i+1:]
					if strings.HasPrefix(st, "not") {
						if UnarySyntax != "()op" {
							panic("Ошибка: недопустимое расположение операндов и операций")
						} else {
							funcName = "not"
							i += 3
						}
					}
				} else if funcName == "" && len(args) > 1 {
					if BinarySyntax != "()op" {
						panic("Ошибка: недопустимое расположение операндов и операций")
					}

					st := expression[i+1:]
					switch {
					case strings.HasPrefix(st, "add"):
						funcName = "add"
						i += 3
					case strings.HasPrefix(st, "sub"):
						funcName = "sub"
						i += 3
					case strings.HasPrefix(st, "pow"):
						funcName = "pow"
						i += 3
					case strings.HasPrefix(st, "div"):
						funcName = "div"
						i += 3
					case strings.HasPrefix(st, "rem"):
						funcName = "rem"
						i += 3
					case strings.HasPrefix(st, "xor"):
						funcName = "xor"
						i += 3
					case strings.HasPrefix(st, "and"):
						funcName = "and"
						i += 3
					case strings.HasPrefix(st, "mult"):
						funcName = "mult"
						i += 4
					case strings.HasPrefix(st, "or"):
						funcName = "or"
						i += 2
					}
				}

				if funcName == "" {
					tokens = append(tokens, fmt.Sprintf("%v", evaluateInfix(args[0])))
					currentToken = ""
				} else {
					for k := range args {
						args[k] = fmt.Sprintf("%v", evaluateInfix(args[k]))
					}
					tokens = append(tokens, executeCommand(funcName, args))
					currentToken = ""
				}
			}
		} else if currentToken != "" && openB != 0 {
			currentToken += string(expression[i])
		} else if currentToken != "" {
			tokens = append(tokens, currentToken)
			tokens = append(tokens, string(expression[i]))
			currentToken = ""
		}
		i++
	}
	if currentToken != "" {
		tokens = append(tokens, currentToken)
	}

	for i := range tokens {
		if val, ok := variables[tokens[i]]; ok {
			tokens[i] = fmt.Sprintf("%v", val)
		}
	}
	return tokens
}


//
func (interp *Interpreter) EvalPostfix(expression interface{}, oper map[string]bool, executeCommand func(string, []string) string) (float64, error) {
	stack := []float64{}

	// Convert expression to tokens based on its type
	var tokens []string
	switch v := expression.(type) {
	case int:
		tokens = strings.Split(strconv.Itoa(v), "")
	case string:
		tokens = strings.Split(v, "")
	default:
		return 0.0, fmt.Errorf("unsupported expression type")
	}

	for _, token := range tokens {
		if interp.IsNumber(token) {
			num, err := strconv.ParseFloat(token, 64)
			if err != nil {
				return 0.0, err
			}
			stack = append(stack, num)
		} else if oper[token] {
			if len(stack) < 2 {
				return 0.0, fmt.Errorf("invalid expression")
			}
			operand2 := stack[len(stack)-1]
			operand1 := stack[len(stack)-2]
			stack = stack[:len(stack)-2] // Pop two elements from stack

			// Execute the operation
			args := []string{fmt.Sprintf("%v", operand1), fmt.Sprintf("%v", operand2)}
			result := executeCommand(token, args)
			numResult, err := strconv.ParseFloat(result, 64)
			if err != nil {
				return 0.0, err
			}
			stack = append(stack, numResult)
		} else {
			return 0.0, fmt.Errorf("invalid token: %v", token)
		}
	}

	if len(stack) != 1 {
		return 0.0, fmt.Errorf("invalid expression")
	}

	return stack[0], nil
}

//
func split2str(s, sep string, n int) []string {
	parts := strings.SplitN(s, sep, n)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}


func (interp *Interpreter) ExecuteCommand(cmd string, args []string) int {
	if cmd == "not" {
		arg := interp.EvaluateExpression(args[0])
		return ^arg & 0xFFFFFFFF
	}

	arg1 := interp.EvaluateExpression(args[0])
	arg2 := interp.EvaluateExpression(args[1])

	switch cmd {
	case "add":
		return (arg1 + arg2) & 0xFFFFFFFF
	case "mult":
		return (arg1 * arg2) & 0xFFFFFFFF
	case "sub":
		return (arg1 - arg2) & 0xFFFFFFFF
	case "div":
		return arg1 / arg2
	case "rem":
		return arg1 % arg2
	case "xor":
		return arg1 ^ arg2
	case "and":
		return arg1 & arg2
	case "or":
		return arg1 | arg2
	case "pow":
		return int(math.Pow(float64(arg1), float64(arg2))) & 0xFFFFFFFF
	default:
		return 0
	}
}


func (interp *Interpreter) RomanToInt(s string) int {
	romanNumerals := map[rune]int{
		'I': 1, 'V': 5, 'X': 10, 'L': 50,
		'C': 100, 'D': 500, 'M': 1000,
	}

	total := 0
	prevValue := 0

	for i := len(s) - 1; i >= 0; i-- {
		value := romanNumerals[rune(s[i])]
		if value < prevValue {
			total -= value
		} else {
			total += value
		}
		prevValue = value
	}

	return total
}

func (interp *Interpreter) FibSequence(maxValue int) []int {
	fibs := []int{1, 2}

	for fibs[len(fibs)-1]+fibs[len(fibs)-2] <= maxValue {
		fibs = append(fibs, fibs[len(fibs)-1]+fibs[len(fibs)-2])
	}

	return fibs
}


func (interp *Interpreter) IsZeckendorf(fibNums, fibs []int) bool {
	fibSet := make(map[int]bool)
	for _, num := range fibNums {
		fibSet[num] = true
	}

	for i := 1; i < len(fibs); i++ {
		if fibSet[fibs[i]] && fibSet[fibs[i-1]] {
			return false
		}
	}

	return true
}



func (interp *Interpreter) ZeckendorfToInt(fibNums []int) int {
	sum := 0
	for _, num := range fibNums {
		sum += num
	}
	return sum
}


func (interp *Interpreter) DebugPrompt() {
	fmt.Println("Доступные команды:")
	fmt.Println("1) Вывод значения и двоичного представления переменной")
	fmt.Println("2) Вывести все переменные")
	fmt.Println("3) Обновить значение существующей переменной")
	fmt.Println("4) Объявить новую переменную")
	fmt.Println("5) Удалить переменную")
	fmt.Println("6) Продолжить выполнение кода")
	fmt.Println("7) Завершить работу интерпретатора")

	for {
		var command string
		fmt.Print("DEBUG> ")
		fmt.Scanln(&command)
		command = strings.TrimSpace(strings.ToLower(command))

		switch command {
		case "1":
			var varName string
			fmt.Print("Введите имя переменной: ")
			fmt.Scanln(&varName)
			value := interp.Variables.Search(varName)
			if value != nil {
				fmt.Printf("%s = %d\n", varName, *value)
				binaryValue := fmt.Sprintf("%032b", *value)
				fmt.Println(strings.Join(splitByWidth(binaryValue, 8), " "))
			} else {
				fmt.Println("Переменная не объявлена")
			}

		case "2":
			for _, varName := range interp.Variables.ObtainAll() {
				value := interp.Variables.Search(varName)
				fmt.Printf("%s = %d\n", varName, *value)
			}

		case "3":
			var varName, hexValue string
			fmt.Print("Введите имя переменной: ")
			fmt.Scanln(&varName)
			if contains(varName, interp.Variables.ObtainAll()) {
				fmt.Print("Введите шестнадцатеричное значение переменной: ")
				fmt.Scanln(&hexValue)
				value, err := strconv.ParseInt(hexValue, 16, 32)
				if err == nil {
					interp.Variables.Insert(varName, int(value))
					fmt.Printf("Значение переменной \"%s\" обновлено\n", varName)
				} else {
					fmt.Println("Некорректное значение")
				}
			} else {
				fmt.Printf("Переменная \"%s\" не объявлена\n", varName)
			}

		case "4":
			var varName, valueType string
			fmt.Print("Введите имя новой переменной: ")
			fmt.Scanln(&varName)
			allVars := interp.variables.ObtainAll()
			for contains(varName, allVars) {
				fmt.Println("Переменная уже объявлена. Введите другое имя переменной.")
				fmt.Print("Введите имя новой переменной: ")
				fmt.Scanln(&varName)
			}

			fmt.Print("Введите тип значения (цекендорфский(1)/римский(2)): ")
			fmt.Scanln(&valueType)

			switch valueType {
			case "1":
				fibSequence := interp.FibSequence(1000000)
				for {
					var fibNums []int
					fmt.Print("Введите число в цекендорфовом представлении: ")
					input := ""
					fmt.Scanln(&input)
					for _, num := range strings.Split(input, " ") {
						n, _ := strconv.Atoi(num)
						fibNums = append(fibNums, n)
					}
					if interp.IsZeckendorf(fibNums, fibSequence) {
						value := interp.ZeckendorfToInt(fibNums)
						interp.variables.Insert(varName, value)
						fmt.Printf("Переменная %s объявлена со значением %d.\n", varName, value)
						break
					} else {
						fmt.Println("Недопустимое цекендорфово представление. Попробуйте снова.")
					}
				}
			case "2":
				var romanValue string
				fmt.Print("Введите значение римскими цифрами: ")
				fmt.Scanln(&romanValue)
				value := interp.RomanToInt(strings.ToUpper(romanValue))
				interp.variables.Insert(varName, value)
				fmt.Printf("Переменная %s объявлена со значением %d.\n", varName, value)
			default:
				fmt.Println("Неизвестный тип значения")
			}

		case "5":
			var varName string
			fmt.Print("Введите имя переменной: ")
			fmt.Scanln(&varName)
			if interp.variables.Search(varName) != nil {
				interp.variables.Delete(varName)
				fmt.Printf("Переменная \"%s\" удалена\n", varName)
			} else {
				fmt.Printf("Переменная \"%s\" не объявлена\n", varName)
			}

		case "6":
			return
		case "7":
			os.Exit(0)
		default:
			fmt.Println("Неизвестная команда")
		}
	}
}

func splitByWidth(s string, width int) []string {
	var result []string
	for i := 0; i < len(s); i += width {
		end := i + width
		if end > len(s) {
			end = len(s)
		}
		result = append(result, s[i:end])
	}
	return result
}









func main() {
	args := os.Args
	if len(args) < 3 {
		fmt.Println("Usage: go run interpreter.go <settings_file> <program_file> [--debug|-d|/debug] [base-assign=<value>] [base-input=<value>] [base-output=<value>]")
		os.Exit(1)
	}

	programFile := args[1]
	settingsFile := args[2]

	baseAssign := 10
	baseInput := 10
	baseOutput := 10

	// Parse command-line arguments for base values
	for _, arg := range args[3:] {
		if strings.HasPrefix(arg, "base-assign") {
			baseAssignStr := strings.Split(arg, "=")[1]
			baseAssign, _ = strconv.Atoi(baseAssignStr)
		} else if strings.HasPrefix(arg, "base-input") {
			baseInputStr := strings.Split(arg, "=")[1]
			baseInput, _ = strconv.Atoi(baseInputStr)
		} else if strings.HasPrefix(arg, "base-output") {
			baseOutputStr := strings.Split(arg, "=")[1]
			baseOutput, _ = strconv.Atoi(baseOutputStr)
		}
	}

	debug := containsAny(args, "--debug", "-d", "/debug")

	// Read program file
	program, err := readFile(programFile)
	if err != nil {
		fmt.Println("Error reading program file:", err)
		os.Exit(1)
	}

	// Create interpreter instance with settings and execute program
	interpreter := NewInterpreter(settingsFile, baseInput, baseOutput, baseAssign, debug)
	interpreter.Execute(program)
}