package build

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"time"
)

type Instruction struct {
	Original    string
	Instruction string
	Args        []string
	CreatedAt   time.Time
}

type Parser struct {
	From         string
	instructions []Instruction
}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Parse(path string) ([]Instruction, error) {
	cfg, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	for _, line := range bytes.Split(cfg, []byte("\n")) {
		strLine := strings.TrimSpace(string(line))
		sp := strings.SplitN(strLine, " ", 2)
		switch strings.ToUpper(strings.TrimSpace(sp[0])) {
		case "FROM":
			p.From = strings.TrimSpace(sp[1])
			p.instructions = append(p.instructions, Instruction{
				Original:    strLine,
				Instruction: "FROM",
				Args:        []string{strings.TrimSpace(sp[1])},
			})
		case "ENV":
			p.instructions = append(p.instructions, Instruction{
				Original:    strLine,
				Instruction: "ENV",
				Args:        []string{strings.TrimSpace(sp[1])},
			})
		case "COPY":
			if copy := strings.SplitN(sp[1], " ", 2); len(copy) == 2 {
				p.instructions = append(p.instructions, Instruction{
					Original:    strLine,
					Instruction: "COPY",
					Args:        []string{strings.TrimSpace(copy[0]), strings.TrimSpace(copy[1])},
				})
			}
		case "WORKDIR":
			p.instructions = append(p.instructions, Instruction{
				Original:    strLine,
				Instruction: "WORKDIR",
				Args:        []string{strings.TrimSpace(sp[1])},
			})
		case "ENTRYPOINT":
			var result []string
			if err := json.Unmarshal([]byte(sp[1]), &result); err == nil {
				p.instructions = append(p.instructions, Instruction{
					Original:    strLine,
					Instruction: "ENTRYPOINT",
					Args:        result,
				})
			} else {
				return nil, err
			}
		case "EXPOSE":
			args := []string{}
			for _, port := range strings.Split(strings.TrimSpace(sp[1]), " ") {
				portStr := strings.TrimSpace(port)
				if strings.Contains(portStr, "/") {
					args = append(args, portStr)
				} else {
					args = append(args, portStr+"/tcp")
				}
			}
			if len(args) > 0 {
				p.instructions = append(p.instructions, Instruction{
					Original:    strLine,
					Instruction: "EXPOSE",
					Args:        args,
				})
			}
		case "CMD":
			var result []string
			if err := json.Unmarshal([]byte(sp[1]), &result); err == nil {
				p.instructions = append(p.instructions, Instruction{
					Original:    strLine,
					Instruction: "CMD",
					Args:        result,
				})
			} else {
				return nil, err
			}
		}
	}

	return p.instructions, nil
}

func (p *Parser) GetFrom() string {
	return p.From
}

func (p *Parser) GetInstructions() []Instruction {
	return p.instructions
}
