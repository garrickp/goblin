package main

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"unicode"
)

type ScaledNumber struct {
	Number int64
	Scale  int32
}

type CommandLine string

type StackItem struct {
	IsNumber bool
	Number   ScaledNumber
	Command  CommandLine
}

type Stack struct {
	Items   []*StackItem
	Pointer int
}

const StackGrowthFactor float64 = 1.2
const MemLogLen int = 6144
const DebugLog bool = false

var RootStack *Stack
var Registers map[rune]*Stack
var InterpScale int32 = 0
var InterpLevel int32 = 0
var OutBase int64 = 10
var OutScale int32 = 0
var MemLog []string
var MemLogP int = 0

// Record logging messages into a rotating memory buffer. We can potentially
// print these out by setting DebugLog to true, or if we need to GDB in, we can
// look at the contents of MemLog. Shouldn't affect our run time *too* much
func Log(msg string, args ...interface{}) {
	MemLog[MemLogP] = fmt.Sprintf(msg, args...)
	MemLogP += 1
	if MemLogP >= MemLogLen {
		MemLogP = 0
	}
}

// Print out the log messages. If the buffer has rotated past MemLogLen, we
// won't output all messages, but they can still be captured using GDB
func LogOut() {
	if DebugLog {
		for i := 0; i < MemLogP; i++ {
			fmt.Println(MemLog[i])
		}
	}
}

// Create a new StackItem, initialized to being a number of 0.
func NewItem() (i *StackItem) {
	Log("creating new stack item")
	i = new(StackItem)
	i.IsNumber = true
	i.Number = ScaledNumber{Number: 0, Scale: 0}
	return i
}

// Create a new stack, containing a default item
func NewStack() (s *Stack) {
	Log("creating new stack")
	s = new(Stack)
	s.Pointer = 0
	s.Items = make([]*StackItem, 100)
	s.Items[0] = NewItem()
	return s
}

// Basic push functionality on the stack. Grows the stack by
// `StackGrowthFactor` if there is not enough capacity
func (s *Stack) Push(i *StackItem) {
	Log("pushing onto stack: %v", i)
	// Grow if necessary
	if s.Pointer+1 > cap(s.Items) {
		swapArray := make([]*StackItem, int(float64(cap(s.Items))*StackGrowthFactor))
		s.Items, swapArray = swapArray, s.Items
		copy(swapArray, s.Items)
	}

	// Add the item to the array
	s.Pointer += 1
	s.Items[s.Pointer] = i
}

// Remove the last item from the stack & return it
func (s *Stack) Pop() (i *StackItem) {
	i = s.Items[s.Pointer]
	s.Pointer -= 1
	Log("popped off stack: %v", i)

	// Ensure that an item with the number 0 remains at all times
	if s.Pointer < 0 {
		s.Push(NewItem())
	}

	return
}

// Return a copy of the item from the top of the stack
func (s *Stack) Peek() (i *StackItem) {
	j = s.Items[s.Pointer]

	i := NewItem()
	i.IsNumber = j.IsNumber
	i.Number = j.Number
	i.Command = j.Command

	Log("peeked at stack: %v", i)
	return
}

func IntPower(n int64, y int32) int64 {
	m := n
	if y > 0 {
		for i:=1; i < y; i++ {
			m *= n
		}
	} else {
		for i:=0; i >= y; i-- {
			m /= n
		}
	}
	return m
}

// Maintains integer values without while scaling the number up or down.
func RescaleNumber(n *ScaledNumber, newScale int32) {
	Log("rescaling number: n=%v, s=%v", n, newScale)
	n.Number = n.Number * intPower(10, newScale-n.Scale)
}

func IntMax(na ...int32) int32 {
	var max int32 = math.MinInt32
	for _, x := range na {
		if x > max {
			max = x
		}
	}
	return max
}

func IntMin(na ...int32) int32 {
	var min int32 = math.MaxInt32
	for _, x := range na {
		if x < min {
			min = x
		}
	}
	return min
}

func IntAbs(n int32) int32 {
	if n >= 0 {
		return n
	} else {
		return n * -1
	}
}

func ReadNumber(reader *bufio.Reader) (item *StackItem, err error) {
	item = NewItem()

	var next rune
	var numStr string = ""
	var scale int32 = 0
	var base int = 10
	var eofReached bool = false

	for {
		next, _, err = reader.ReadRune()
		if err != nil {
			if err == io.EOF {
				eofReached = true
				break
			}
			return
		}

		if next == '_' {
			numStr += string('-')
			continue
		}

		if next == '.' {
			scale += 1
			continue
		}

		if unicode.In(next, unicode.Hex_Digit) {
			if !unicode.In(next, unicode.Digit) {
				base = 16
			}
			if scale > 0 {
				scale += 1
			}
			numStr += string(next)

		} else {
			break
		}
	}

	// Replace the next character if we did not use it, and we're not at the end of the file
	if !eofReached {
		if err = reader.UnreadRune(); err != nil {
			return
		}
	}

	item.Number.Number, err = strconv.ParseInt(numStr, base, 64)
	if err != nil {
		return
	}

	item.Number.Scale = scale
	item.IsNumber = true

	Log("read number: %v", item)

	return
}

func ReadCommand(reader *bufio.Reader) (item *StackItem, err error) {
	item = NewItem()

	var next rune
	var command string = ""
	var startFound bool = false
	var endFound bool = false

	for {
		next, _, err = reader.ReadRune()
		if err != nil {
			if err == io.EOF {
				break
			}
			return
		}

		if next == '[' {
			startFound = true
			continue
		} else if !startFound {
			err = fmt.Errorf("Expected '[', found %v", next)
			return
		}

		if next == ']' {
			endFound = true
			break
		}

		command += string(next)

	}

	if !endFound {
		err = fmt.Errorf("Expected ']', not found")
		return
	}

	// Since the last character was the ']' and belongs to this command, we
	// don't need to push anything back onto the reader like we do with the ReadNumber

	item.IsNumber = false
	item.Command = CommandLine(command)

	Log("read command: %v", item)

	return

}

// Base interpreter which reads through the input stream & executes the
// provided commands
func Interp(r io.Reader) error {
	reader := bufio.NewReader(r)

	for {
		next, _, readErr := reader.ReadRune()
		Log("read next: %c", next)
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return readErr
		}

		if unicode.In(next, unicode.Digit) || next == '_' || next == '.' {
			if err := reader.UnreadRune(); err != nil {
				return err
			}
			if number, err := ReadNumber(reader); err != nil {
				return err
			} else {
				RootStack.Push(number)
				continue
			}
		}

		if next == '[' {
			if err := reader.UnreadRune(); err != nil {
				return err
			}
			if command, err := ReadCommand(reader); err != nil {
				return err
			} else {
				RootStack.Push(command)
				continue
			}
		}

		switch next {
		case '+':
			a := RootStack.Pop()
			b := RootStack.Pop()
			if !a.IsNumber || !b.IsNumber {
				return fmt.Errorf("Expected both items from the stack to be numbers")
			}
			aNum := a.Number
			bNum := b.Number

			targetScale := InterpScale
			if aNum.Scale > bNum.Scale {
				targetScale = aNum.Scale
			} else {
				targetScale = bNum.Scale
			}
			RescaleNumber(&aNum, targetScale)
			RescaleNumber(&bNum, targetScale)

			c := NewItem()
			c.IsNumber = true
			c.Number = ScaledNumber{Scale: targetScale, Number: aNum.Number + bNum.Number}
			RootStack.Push(c)

		case '-':
			a := RootStack.Pop()
			b := RootStack.Pop()
			if !a.IsNumber || !b.IsNumber {
				return fmt.Errorf("Expected both items from the stack to be numbers")
			}
			aNum := a.Number
			bNum := b.Number

			targetScale := InterpScale
			if aNum.Scale > bNum.Scale {
				targetScale = aNum.Scale
			} else {
				targetScale = bNum.Scale
			}
			RescaleNumber(&aNum, targetScale)
			RescaleNumber(&bNum, targetScale)

			c := NewItem()
			c.IsNumber = true
			c.Number = ScaledNumber{Scale: targetScale, Number: aNum.Number - bNum.Number}
			RootStack.Push(c)

		case '/':
			a := RootStack.Pop()
			b := RootStack.Pop()
			if !a.IsNumber || !b.IsNumber {
				return fmt.Errorf("Expected both items from the stack to be numbers")
			}
			aNum := a.Number
			bNum := b.Number

			RescaleNumber(&aNum, InterpScale)
			RescaleNumber(&bNum, InterpScale)

			c := NewItem()
			c.IsNumber = true
			c.Number = ScaledNumber{Scale: InterpScale, Number: aNum.Number / bNum.Number}
			RootStack.Push(c)

		case '*':
			a := RootStack.Pop()
			b := RootStack.Pop()
			if !a.IsNumber || !b.IsNumber {
				return fmt.Errorf("Expected both items from the stack to be numbers")
			}
			aNum := a.Number
			bNum := b.Number

			targetScale := IntMin(aNum.Scale+bNum.Scale, IntMax(InterpScale, aNum.Scale, bNum.Scale))

			RescaleNumber(&aNum, targetScale)
			RescaleNumber(&bNum, targetScale)

			c := NewItem()
			c.IsNumber = true
			c.Number = ScaledNumber{Scale: targetScale, Number: aNum.Number * bNum.Number}
			RootStack.Push(c)

		case '%':
			a := RootStack.Pop()
			b := RootStack.Pop()
			if !a.IsNumber || !b.IsNumber {
				return fmt.Errorf("Expected both items from the stack to be numbers")
			}
			aNum := a.Number
			bNum := b.Number

			targetScale := IntMin(aNum.Scale+bNum.Scale, IntMax(InterpScale, aNum.Scale, bNum.Scale))

			RescaleNumber(&aNum, targetScale)
			RescaleNumber(&bNum, targetScale)

			c := NewItem()
			c.IsNumber = true
			c.Number = ScaledNumber{Scale: targetScale, Number: aNum.Number % bNum.Number}
			RootStack.Push(c)

		case '^':
			a := RootStack.Pop()
			b := RootStack.Pop()
			if !a.IsNumber || !b.IsNumber {
				return fmt.Errorf("Expected both items from the stack to be numbers")
			}
			aNum := a.Number
			bNum := b.Number

			targetScale := IntMin(aNum.Scale*IntAbs(bNum.Scale), IntMax(InterpScale, aNum.Scale))

			RescaleNumber(&aNum, targetScale)
			RescaleNumber(&bNum, 0)

			powNum := aNum
			if bNum.Number >= 0 {
				for i := int64(0); i < bNum.Number; i++ {
					powNum.Number = powNum.Number * aNum.Number
				}
			} else {
				for i := int32(0); i < IntAbs(int32(bNum.Number)); i++ {
					powNum.Number = powNum.Number / aNum.Number
				}
			}

			c := NewItem()
			c.IsNumber = true
			c.Number = ScaledNumber{Scale: targetScale, Number: powNum.Number}
			RootStack.Push(c)

		case 'v':
			// Square Root
		case 's':
			// Store in register
			registerName, _, readErr := reader.ReadRune()
			Log("register name: %c", registerName)
			if readErr != nil {
				return readErr
			}

			registerStack, Ok := Registers[registerName]
			if !Ok {
				registerStack = NewStack()
				Registers[registerName] = registerStack
			}

			registerStack.Pointer = 0
			registerStack.Push(RootStack.Pop())

		case 'S':
			// Push in register
			registerName, _, readErr := reader.ReadRune()
			Log("register name: %c", registerName)
			if readErr != nil {
				return readErr
			}

			registerStack, Ok := Registers[registerName]
			if !Ok {
				registerStack = NewStack()
				Registers[registerName] = registerStack
			}

			registerStack.Push(RootStack.Pop())

		case 'l':
			// Retrieve from register
			registerName, _, readErr := reader.ReadRune()
			Log("register name: %c", registerName)
			if readErr != nil {
				return readErr
			}

			registerStack, Ok := Registers[registerName]
			if !Ok {
				registerStack = NewStack()
				Registers[registerName] = registerStack
				registerStack.Pointer = 0
			} else {
				registerStack.Pointer = 1
			}

			RootStack.Push(registerStack.Pop())
			registerStack.Pointer = 1

		case 'L':
			// Retrieve from top of register stack
			registerName, _, readErr := reader.ReadRune()
			Log("register name: %c", registerName)
			if readErr != nil {
				return readErr
			}

			registerStack, Ok := Registers[registerName]
			if !Ok {
				registerStack = NewStack()
				Registers[registerName] = registerStack
			}

			RootStack.Push(registerStack.Pop())

		case 'd':
			// Duplicate the top item on the stack

			Log("duplicating to stack item")
			a := RootStack.Peek()
			b := NewItem()

			b.IsNumber = a.IsNumber
			b.Number = a.Number
			b.Command = a.Command

			RootStack.Push(b)

		case 'p':
			// TODO: Needs more re-thinking in the case of non-10 output base
			// Print the top item in the stack
			a := RootStack.Peek()
			if a.IsNumber {
				if a.Number.Scale != 0 {
					var aNum float64 = float64(a.Number.Number) / (10.0 * float64(a.Number.Scale))
					fmt.Printf("%f\n", aNum)
				} else {
					fmt.Printf("%d\n", a.Number.Number)
				}
			} else {
				return fmt.Errorf("can not exeucte p on a command")
			}

		case 'P':
			// Pop the top item from the stack & print it as a string
			a := RootStack.Pop()
			if a.IsNumber {
				return fmt.Errorf("P can not execute on a number")
			} else {
				fmt.Printf("%s", string(a.Command))
			}

		case 'f':
			// TODO: Needs more re-thinking in the case of non-10 output base
			// Print out all of the values on the stack
			var output string
			for i := 0; i <= RootStack.Pointer; i++ {
				a := RootStack.Items[i]
				if a.IsNumber {
					if a.Number.Scale != 0 {
						var aNum float64 = float64(a.Number.Number) / (10.0 * float64(a.Number.Scale))
						output = fmt.Sprintf("%f", aNum)
					} else {
						output = fmt.Sprintf("%d", a.Number.Number)
					}
				} else {
					output = string(a.Command)
				}
				fmt.Printf("%s\n", output)
			}

		case 'q':
			// TODO Make this work with the recursive calls to Interp
			InterpLevel -= 2
			if InterpLevel < 0 {
				return nil
			}

		case 'Q':
			// TODO Make this work with the recursive calls to Interp
			dropLevel := RootStack.Pop()
			if dropLevel.IsNumber {
				RescaleNumber(&dropLevel.Number, 0)
				InterpLevel -= int32(dropLevel.Number.Number)
				if InterpLevel < 0 {
					return nil
				}
			} else {
				return fmt.Errorf("Q can not be implemented with a command stack item")
			}

		case 'x':
			cmd := RootStack.Pop()
			if cmd.IsNumber {
				return fmt.Errorf("x can not be implemented with a number")
			}

			sr := strings.NewReader(string(cmd.Command))
			fb := bufio.NewReader(sr)
			err := Interp(fb)

			if err != nil {
				return err
			}

		case 'X':
			x := RootStack.Pop()
			if !x.IsNumber {
				return fmt.Errorf("X can not be implemented with a command stack item")
			}

			x.Number.Number = int64(x.Number.Scale)
			x.Number.Scale = 0

			RootStack.Push(x)

		case '<':
			a := RootStack.Pop()
			b := RootStack.Pop()
			if !a.IsNumber || !b.IsNumber {
				return fmt.Errorf("Expected both items from the stack to be numbers")
			}

			registerName, _, readErr := reader.ReadRune()
			Log("register name: %c", registerName)
			if readErr != nil {
				return readErr
			}
			var aNum float64 = float64(a.Number.Number) / math.Pow(10.0, float64(a.Number.Scale))
			var bNum float64 = float64(b.Number.Number) / math.Pow(10.0, float64(b.Number.Scale))

			if aNum < bNum {
				cmd := Registers[registerName].Peek()
				if cmd.IsNumber {
					return fmt.Errorf("x can not be implemented with a number")
				}

				sr := strings.NewReader(string(cmd.Command))
				fb := bufio.NewReader(sr)
				err := Interp(fb)

				if err != nil {
					return err
				}
			}

		case '>':
			a := RootStack.Pop()
			b := RootStack.Pop()
			if !a.IsNumber || !b.IsNumber {
				return fmt.Errorf("Expected both items from the stack to be numbers")
			}

			registerName, _, readErr := reader.ReadRune()
			Log("register name: %c", registerName)
			if readErr != nil {
				return readErr
			}
			var aNum float64 = float64(a.Number.Number) / math.Pow(10.0, float64(a.Number.Scale))
			var bNum float64 = float64(b.Number.Number) / math.Pow(10.0, float64(b.Number.Scale))
			Log("%v > %v", aNum, bNum)

			if aNum > bNum {
				cmd := Registers[registerName].Peek()
				Log("cmd from register: %v", cmd)
				if cmd.IsNumber {
					return fmt.Errorf("x can not be implemented with a number")
				}

				sr := strings.NewReader(string(cmd.Command))
				fb := bufio.NewReader(sr)
				err := Interp(fb)

				if err != nil {
					return err
				}
			}

		case '=':
			a := RootStack.Pop()
			b := RootStack.Pop()
			if !a.IsNumber || !b.IsNumber {
				return fmt.Errorf("Expected both items from the stack to be numbers")
			}

			registerName, _, readErr := reader.ReadRune()
			Log("register name: %c", registerName)
			if readErr != nil {
				return readErr
			}

			if a.Number.Scale == b.Number.Scale && a.Number.Number == b.Number.Number {
				cmd := Registers[registerName].Peek()
				if cmd.IsNumber {
					return fmt.Errorf("x can not be implemented with a number")
				}

				sr := strings.NewReader(string(cmd.Command))
				fb := bufio.NewReader(sr)
				err := Interp(fb)

				if err != nil {
					return err
				}
			}

		case '!':
			// Execute a bash command up to the newline

		case 'c':
			RootStack = NewStack()

		case 'i':
			a := RootStack.Pop()
			if !a.IsNumber {
				return fmt.Errorf("i can not interpret a command as a scale")
			}
			RescaleNumber(&a.Number, 0)
			InterpScale = int32(a.Number.Number)

		case 'I':
			i := NewItem()
			i.IsNumber = true
			i.Number.Number = int64(InterpScale)

			RootStack.Push(i)

		case 'o':
			// In bases larger than 10, each `digit' prints as a group of decimal digits.
			a := RootStack.Pop()
			if !a.IsNumber {
				return fmt.Errorf("o can not interpret a command as a scale")
			}
			RescaleNumber(&a.Number, 0)
			OutBase = a.Number.Number

		case 'O':
			i := NewItem()
			i.IsNumber = true
			i.Number.Number = OutBase

			RootStack.Push(i)

		case 'k':
			a := RootStack.Pop()
			if !a.IsNumber {
				return fmt.Errorf("k can not interpret a command as a scale")
			}
			RescaleNumber(&a.Number, 0)
			OutScale = int32(a.Number.Number)

		case 'z':
			i := NewItem()
			i.IsNumber = true
			i.Number.Number = int64(RootStack.Pointer)

			RootStack.Push(i)

		case 'Z':
			// TODO Replace the number on the top of the stack with its length.

		case '?':
			// A line of input is taken from the input source (usually the terminal) and executed.
			// TODO Figure out how this work when commands from stdin is already being read from by default

		}
	}
	return nil
}

func main() {
	// Initialize our memory
	MemLog = make([]string, MemLogLen)
	RootStack = NewStack()
	Registers = make(map[rune]*Stack)

	if len(os.Args) > 1 {
		Log("opening file: %v", os.Args[1])
		file, err := os.Open(os.Args[1])
		if err != nil {
			LogOut()
			panic(err.Error())
		}
		Log("interpreting file")
		interpErr := Interp(file)
		if interpErr != nil {
			LogOut()
			panic(interpErr.Error())
		}
	} else {
		Log("interpreting stdin")
		interpErr := Interp(os.Stdin)
		if interpErr != nil {
			LogOut()
			panic(interpErr.Error())
		}
	}
	LogOut()

}
