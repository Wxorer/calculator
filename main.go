package main

import (
	"errors"
	"strconv"
	"strings"
)

// Calc вычисляет значение математического выражения
func Calc(expression string) (float64, error) {
	// Удаляем пробелы из выражения
	expression = strings.ReplaceAll(expression, " ", "")

	if len(expression) == 0 {
		return 0, errors.New("пустое выражение")
	}

	// Преобразуем выражение в токены (числа и операторы)
	tokens, err := parseExpression(expression)
	if err != nil {
		return 0, err
	}

	// Преобразуем инфиксную запись в постфиксную
	postfix, err := toPostfix(tokens)
	if err != nil {
		return 0, err
	}

	// Вычисляем результат
	return calculatePostfix(postfix)
}

// Структура для хранения токена (число или оператор)
type token struct {
	value    string
	isNumber bool
}

// parseExpression разбирает строку на токены
func parseExpression(expr string) ([]token, error) {
	var tokens []token
	var currentNumber string

	// Обработка отрицательных чисел в начале выражения или после открывающей скобки
	if expr[0] == '-' {
		expr = "0" + expr
	}
	expr = strings.ReplaceAll(expr, "(-", "(0-")

	for i := 0; i < len(expr); i++ {
		char := expr[i]

		switch {
		case char >= '0' && char <= '9' || char == '.':
			currentNumber += string(char)

		case char == '+' || char == '-' || char == '*' || char == '/' || char == '(' || char == ')':
			if currentNumber != "" {
				// Преобразуем строку в число и проверяем корректность
				if _, err := strconv.ParseFloat(currentNumber, 64); err != nil {
					return nil, errors.New("некорректное число: " + currentNumber)
				}
				tokens = append(tokens, token{currentNumber, true})
				currentNumber = ""
			}
			tokens = append(tokens, token{string(char), false})

		default:
			return nil, errors.New("неподдерживаемый символ: " + string(char))
		}
	}

	if currentNumber != "" {
		if _, err := strconv.ParseFloat(currentNumber, 64); err != nil {
			return nil, errors.New("некорректное число: " + currentNumber)
		}
		tokens = append(tokens, token{currentNumber, true})
	}

	return tokens, nil
}

// Возвращает приоритет оператора
func priority(op string) int {
	switch op {
	case "+", "-":
		return 1
	case "*", "/":
		return 2
	default:
		return 0
	}
}

// toPostfix преобразует инфиксную запись в постфиксную (обратную польскую нотацию)
func toPostfix(tokens []token) ([]token, error) {
	var result []token
	var stack []string

	for _, t := range tokens {
		if t.isNumber {
			result = append(result, t)
			continue
		}

		switch t.value {
		case "(":
			stack = append(stack, t.value)

		case ")":
			for len(stack) > 0 && stack[len(stack)-1] != "(" {
				result = append(result, token{stack[len(stack)-1], false})
				stack = stack[:len(stack)-1]
			}
			if len(stack) == 0 {
				return nil, errors.New("несбалансированные скобки")
			}
			// Удаляем открывающую скобку
			stack = stack[:len(stack)-1]

		default: // операторы
			for len(stack) > 0 && stack[len(stack)-1] != "(" &&
				priority(stack[len(stack)-1]) >= priority(t.value) {
				result = append(result, token{stack[len(stack)-1], false})
				stack = stack[:len(stack)-1]
			}
			stack = append(stack, t.value)
		}
	}

	// Добавляем оставшиеся операторы
	for len(stack) > 0 {
		if stack[len(stack)-1] == "(" {
			return nil, errors.New("несбалансированные скобки")
		}
		result = append(result, token{stack[len(stack)-1], false})
		stack = stack[:len(stack)-1]
	}

	return result, nil
}

// calculatePostfix вычисляет значение выражения в постфиксной записи
func calculatePostfix(tokens []token) (float64, error) {
	var stack []float64

	for _, t := range tokens {
		if t.isNumber {
			num, _ := strconv.ParseFloat(t.value, 64)
			stack = append(stack, num)
			continue
		}

		if len(stack) < 2 {
			return 0, errors.New("некорректное выражение")
		}
		// Берем два последних числа из стека
		b := stack[len(stack)-1]
		a := stack[len(stack)-2]
		stack = stack[:len(stack)-2]

		// Выполняем операцию
		var result float64
		switch t.value {
		case "+":
			result = a + b
		case "-":
			result = a - b
		case "*":
			result = a * b
		case "/":
			if b == 0 {
				return 0, errors.New("деление на ноль")
			}
			result = a / b
		}

		stack = append(stack, result)
	}

	if len(stack) != 1 {
		return 0, errors.New("некорректное выражение")
	}

	return stack[0], nil
}
