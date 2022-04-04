package expect

import (
  "bytes"
  "strings"
  "io"
  "errors"
)

type Cmd struct {
  Index int
  Type string
  Data string
  Level int
  Visited bool
}

func or(in []Cmd, lvl int) (out []Cmd) {
  for _, cmd := range in {
    if cmd.Type == "expect" && cmd.Level == lvl {
      out = append(out, cmd)
    }
  }
  return
}

type ExpectModule struct {
  Cmds []Cmd
  Step int
  Level int
}

func NewExpectModule(input string) (*ExpectModule, error) {
  cmds, err := lex([]byte(input))
  if err != nil {
    return nil, err
  }
  return &ExpectModule{
    Step: 0,
    Level: 0,
    Cmds: cmds,
  }, nil
}

func (exp *ExpectModule) Next(input string) (output string, write bool) {
  for _, cmd := range exp.Cmds[exp.Step:] {
    if cmd.Visited {
      continue
    }
    if exp.Level == cmd.Level && exp.Step + 1 == cmd.Index || (exp.Level > cmd.Level) {
      exp.Level = cmd.Level
      if cmd.Type == "expect" && strings.Contains(input, cmd.Data) {
        cmd.Visited = true
        exp.Step = cmd.Index
      } else if cmd.Type == "send" {
        cmd.Visited = true
        exp.Step = cmd.Index
        return cmd.Data, true
      }
      return
    } else if exp.Level < cmd.Level {
      if cmd.Type == "expect" {
        visited := false
        for _, cond := range or(exp.Cmds, cmd.Level) {
          if strings.Contains(input, cond.Data) {
            exp.Step = cond.Index
            exp.Level += 2
            visited = true
            break
          }
        }
        if visited {
          for _, cond := range or(exp.Cmds, cmd.Level) {
            exp.Cmds[cond.Index-1].Visited = true
          }
        }
      }
      return
    }
  }
  return
}

func lex(in []byte) (cmds []Cmd, err error) {
  buf := bytes.NewBuffer(in)
  lvl := 0
  cmd := Cmd{Index: 1}
  stack := []Cmd{}
  accumulator := ""
  for {
    run, err := buf.ReadString(' ')
    if err != nil {
      if err == io.EOF {
        break
      }
      return nil, err
    }
    run = strings.Trim(accumulator + run, " ")
    switch true {
      case run == "{":
        lvl++
        cmd.Level = lvl
        if cmd.Type != "" {
         stack = append(stack, cmd)
        }
        accumulator = ""
      case run == "}":
        lvl--
        cmd.Level = lvl
        if len(stack) > 0 {
          cmd = stack[len(stack)-1]
          stack = stack[:len(stack)-1]
        }
        accumulator = ""
      case run == "send", run == "expect":
        cmd.Type = run
      case strings.HasSuffix(run, ";"):
        if len(run)-2 < 1 {
          return nil, errors.New("Invalid expression")
        }
        cmd.Data = run[1:len(run)-2]
        cmd.Index = len(cmds) + 1
        cmds = append(cmds, cmd)
        cmd = Cmd{Level: lvl, Index: len(cmds) + 1}
        accumulator = ""
      case run[0] == run[len(run)-1] && run[0] == []byte("\"")[0]:
        if len(run)-1 < 1 {
          return nil, errors.New("Invalid expression")
        }
        cmd.Data = run[1:len(run)-1]
        cmd.Index = len(cmds) + 1
        cmds = append(cmds, cmd)
        cmd = Cmd{Level: lvl}
        accumulator = ""
      default:
        accumulator += run
    }
  }
  return cmds, nil
}





