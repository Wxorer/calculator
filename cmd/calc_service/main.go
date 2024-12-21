package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type CalculateRequest struct {
	Expression string `json:"expression"`
}

type CalculateResponse struct {
	Result float64 `json:"result,omitempty"`
	Error  string  `json:"error,omitempty"`
}

func Calc(expression string) (float64, error) {
	expression = strings.ReplaceAll(expression, " ", "")

	if len(expression) == 0 {
		return 0, errors.New("пустое выражение")
	}

	tokens, err := parseExpression(expression)
	if err != nil {
		return 0, err
	}

	postfix, err := toPostfix(tokens)
	if err != nil {
		return 0, err
	}

	return calculatePostfix(postfix)
}

type token struct {
	value    string
	isNumber bool
}

func parseExpression(expr string) ([]token, error) {
	var tokens []token
	var currentNumber string

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
			stack = stack[:len(stack)-1]

		default:
			for len(stack) > 0 && stack[len(stack)-1] != "(" &&
				priority(stack[len(stack)-1]) >= priority(t.value) {
				result = append(result, token{stack[len(stack)-1], false})
				stack = stack[:len(stack)-1]
			}
			stack = append(stack, t.value)
		}
	}

	for len(stack) > 0 {
		if stack[len(stack)-1] == "(" {
			return nil, errors.New("несбалансированные скобки")
		}
		result = append(result, token{stack[len(stack)-1], false})
		stack = stack[:len(stack)-1]
	}

	return result, nil
}

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

		b := stack[len(stack)-1]
		a := stack[len(stack)-2]
		stack = stack[:len(stack)-2]

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

func calculateHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	requestID := fmt.Sprintf("%d", startTime.UnixNano())

	log.Printf("[%s] Получен новый запрос %s %s", requestID, r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		log.Printf("[%s] Метод не разрешен: %s", requestID, r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[%s] Ошибка при чтении тела запроса: %v", requestID, err)
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprint(w, "Expression is not valid")
		return
	}
	expression := string(body)

	log.Printf("[%s] Получено выражение для вычисления: %s", requestID, expression)

	result, err := Calc(expression)
	if err != nil {
		switch err.Error() {
		case "неподдерживаемый символ", "некорректное число", "некорректное выражение", "несбалансированные скобки", "пустое выражение":
			log.Printf("[%s] Ошибка валидации: %v", requestID, err)
			w.WriteHeader(http.StatusUnprocessableEntity)
			fmt.Fprint(w, "Expression is not valid")
		default:
			log.Printf("[%s] Внутренняя ошибка: %v", requestID, err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "Internal server error")
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%v", result)

	duration := time.Since(startTime)
	log.Printf("[%s] Запрос обработан успешно за %v. Результат: %v", requestID, duration, result)
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.Printf("Запуск сервера на порту :8080")

	http.HandleFunc("/api/v1/calculate", calculateHandler)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
