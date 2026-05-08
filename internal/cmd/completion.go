package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

func runCompletion(args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Fprintln(os.Stderr, "Generate shell completions")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage: jmcp completion bash|zsh|fish")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Examples:")
		fmt.Fprintln(os.Stderr, "  jmcp completion bash > ~/.bash_completion.d/jmcp")
		fmt.Fprintln(os.Stderr, "  jmcp completion zsh  > ~/.zfunc/_jmcp")
		fmt.Fprintln(os.Stderr, "  jmcp completion fish > ~/.config/fish/completions/jmcp.fish")
		if args == nil || len(args) == 0 {
			return errors.New("shell argument required")
		}
		return errHelp
	}
	switch args[0] {
	case "bash":
		return writeBashCompletion(os.Stdout)
	case "zsh":
		return writeZshCompletion(os.Stdout)
	case "fish":
		return writeFishCompletion(os.Stdout)
	default:
		return fmt.Errorf("unsupported shell %q, use bash, zsh, or fish", args[0])
	}
}

var commands = []struct {
	name string
	desc string
}{
	{"health", "Check Jaeger server health"},
	{"services", "List available services"},
	{"spans", "List span names for a service"},
	{"search", "Search traces for a service"},
	{"topology", "Show trace topology tree"},
	{"errors", "Show trace error spans"},
	{"details", "Show span details"},
	{"critical-path", "Show trace critical path"},
	{"deps", "Show service dependencies"},
	{"investigate", "Full trace investigation"},
	{"report", "Generate trace investigation report"},
	{"snapshot", "Capture service trace snapshot"},
	{"diff", "Compare two snapshots"},
	{"watch", "Watch for new traces"},
	{"blame", "Root cause analysis"},
	{"export", "Export trace data"},
	{"check", "Health gate check for CI/CD"},
	{"init", "Initialize .jmcp project directory"},
	{"replay", "Replay investigation from report"},
	{"completion", "Generate shell completions"},
}

func writeBashCompletion(w io.Writer) error {
	cmdNames := make([]string, len(commands))
	for i, c := range commands {
		cmdNames[i] = c.name
	}
	_, err := fmt.Fprintf(w, `_jmcp() {
    local cur prev cmds global_flags
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    cmds="%s"
    global_flags="--endpoint --output --save --timeout --no-color --verbose --help --version"

    if [[ ${COMP_CWORD} -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "${cmds} ${global_flags}" -- "${cur}") )
        return 0
    fi

    case "${prev}" in
        --endpoint|-e)
            COMPREPLY=( $(compgen -W "http://localhost:16687/mcp" -- "${cur}") )
            return 0
            ;;
        --output|-o)
            COMPREPLY=( $(compgen -W "table json raw" -- "${cur}") )
            return 0
            ;;
        --save|-s)
            COMPREPLY=( $(compgen -f -- "${cur}") )
            return 0
            ;;
        --format)
            COMPREPLY=( $(compgen -W "json csv dot md" -- "${cur}") )
            return 0
            ;;
    esac

    local cmd=""
    for ((i=1; i < COMP_CWORD; i++)); do
        case "${COMP_WORDS[i]}" in
            -*) ;;
            *) cmd="${COMP_WORDS[i]}"; break ;;
        esac
    done

    case "${cmd}" in
        search)
            COMPREPLY=( $(compgen -W "--since --until --errors --depth --span --min-duration --max-duration --attr" -- "${cur}") )
            ;;
        watch)
            COMPREPLY=( $(compgen -W "--interval --errors --alert --since" -- "${cur}") )
            ;;
        check)
            COMPREPLY=( $(compgen -W "--error-rate --p95 --since --depth" -- "${cur}") )
            ;;
        snapshot)
            COMPREPLY=( $(compgen -W "--label --errors --since --depth --dir" -- "${cur}") )
            ;;
        report)
            COMPREPLY=( $(compgen -W "--dir --title --format --note" -- "${cur}") )
            ;;
        export)
            COMPREPLY=( $(compgen -W "--format" -- "${cur}") )
            ;;
        topology|investigate)
            COMPREPLY=( $(compgen -W "--depth" -- "${cur}") )
            ;;
        services)
            COMPREPLY=( $(compgen -W "--pattern --limit" -- "${cur}") )
            ;;
        spans)
            COMPREPLY=( $(compgen -W "--kind --pattern --limit" -- "${cur}") )
            ;;
        deps)
            COMPREPLY=( $(compgen -W "--since --until" -- "${cur}") )
            ;;
        completion)
            COMPREPLY=( $(compgen -W "bash zsh fish" -- "${cur}") )
            ;;
        *)
            COMPREPLY=( $(compgen -W "${global_flags}" -- "${cur}") )
            ;;
    esac
    return 0
}
complete -F _jmcp jmcp
`, strings.Join(cmdNames, " "))
	return err
}

func writeZshCompletion(w io.Writer) error {
	_, err := fmt.Fprintf(w, `#compdef jmcp

_jmcp() {
    local -a commands
    commands=(
`)
	if err != nil {
		return err
	}
	for _, c := range commands {
		fmt.Fprintf(w, "        '%s:%s'\n", c.name, c.desc)
	}
	_, err = fmt.Fprint(w, `    )

    local -a global_flags
    global_flags=(
        '(-e --endpoint)'{-e,--endpoint}'[MCP endpoint URL]:url:'
        '(-o --output)'{-o,--output}'[Output format]:format:(table json raw)'
        '(-s --save)'{-s,--save}'[Save output to file]:file:_files'
        '(-t --timeout)'{-t,--timeout}'[Request timeout]:duration:'
        '(-v --verbose)'{-v,--verbose}'[Log protocol details]'
        '--no-color[Disable color output]'
        '--help[Show help]'
        '--version[Show version]'
    )

    _arguments -C \
        $global_flags \
        '1:command:->command' \
        '*::arg:->args'

    case $state in
        command)
            _describe -t commands 'jmcp command' commands
            ;;
        args)
            case ${words[1]} in
                search)
                    _arguments \
                        '--since[Lookback duration]:duration:' \
                        '--until[End time]:time:' \
                        '--errors[Only error traces]' \
                        '--depth[Search depth]:number:' \
                        '--span[Span name]:name:' \
                        '--min-duration[Min duration]:duration:' \
                        '--max-duration[Max duration]:duration:' \
                        '*--attr[Attribute KEY=VALUE]:attr:' \
                        '1:service:'
                    ;;
                watch)
                    _arguments \
                        '--interval[Poll interval]:duration:' \
                        '--errors[Only error traces]' \
                        '--alert[Terminal bell on findings]' \
                        '--since[Initial lookback]:duration:' \
                        '1:service:'
                    ;;
                check)
                    _arguments \
                        '--error-rate[Max error percentage]:percent:' \
                        '--p95[Max p95 latency]:duration:' \
                        '--since[Lookback window]:duration:' \
                        '--depth[Sample size]:number:' \
                        '1:service:'
                    ;;
                completion)
                    _arguments '1:shell:(bash zsh fish)'
                    ;;
                *)
                    _arguments '1:trace-id:'
                    ;;
            esac
            ;;
    esac
}

_jmcp "$@"
`)
	return err
}

func writeFishCompletion(w io.Writer) error {
	_, err := fmt.Fprint(w, "# jmcp fish completions\n\n")
	if err != nil {
		return err
	}
	for _, c := range commands {
		fmt.Fprintf(w, "complete -c jmcp -n '__fish_use_subcommand' -a '%s' -d '%s'\n", c.name, c.desc)
	}
	fmt.Fprint(w, `
# Global flags
complete -c jmcp -l endpoint -s e -d 'MCP endpoint URL' -x
complete -c jmcp -l output -s o -d 'Output format' -xa 'table json raw'
complete -c jmcp -l save -s s -d 'Save output to file' -rF
complete -c jmcp -l timeout -s t -d 'Request timeout' -x
complete -c jmcp -l verbose -s v -d 'Log protocol details'
complete -c jmcp -l no-color -d 'Disable color output'
complete -c jmcp -l help -d 'Show help'
complete -c jmcp -l version -d 'Show version'

# search flags
complete -c jmcp -n '__fish_seen_subcommand_from search' -l since -d 'Lookback duration' -x
complete -c jmcp -n '__fish_seen_subcommand_from search' -l until -d 'End time' -x
complete -c jmcp -n '__fish_seen_subcommand_from search' -l errors -d 'Only error traces'
complete -c jmcp -n '__fish_seen_subcommand_from search' -l depth -d 'Search depth' -x
complete -c jmcp -n '__fish_seen_subcommand_from search' -l span -d 'Span name' -x
complete -c jmcp -n '__fish_seen_subcommand_from search' -l min-duration -d 'Min duration' -x
complete -c jmcp -n '__fish_seen_subcommand_from search' -l max-duration -d 'Max duration' -x
complete -c jmcp -n '__fish_seen_subcommand_from search' -l attr -d 'Attribute KEY=VALUE' -x

# watch flags
complete -c jmcp -n '__fish_seen_subcommand_from watch' -l interval -d 'Poll interval' -x
complete -c jmcp -n '__fish_seen_subcommand_from watch' -l errors -d 'Only error traces'
complete -c jmcp -n '__fish_seen_subcommand_from watch' -l alert -d 'Terminal bell'
complete -c jmcp -n '__fish_seen_subcommand_from watch' -l since -d 'Initial lookback' -x

# check flags
complete -c jmcp -n '__fish_seen_subcommand_from check' -l error-rate -d 'Max error percentage' -x
complete -c jmcp -n '__fish_seen_subcommand_from check' -l p95 -d 'Max p95 latency' -x
complete -c jmcp -n '__fish_seen_subcommand_from check' -l since -d 'Lookback window' -x
complete -c jmcp -n '__fish_seen_subcommand_from check' -l depth -d 'Sample size' -x

# export flags
complete -c jmcp -n '__fish_seen_subcommand_from export' -l format -d 'Export format' -xa 'json csv dot'

# report flags
complete -c jmcp -n '__fish_seen_subcommand_from report' -l dir -d 'Report directory' -x
complete -c jmcp -n '__fish_seen_subcommand_from report' -l title -d 'Report title' -x
complete -c jmcp -n '__fish_seen_subcommand_from report' -l format -d 'Report format' -xa 'md json'
complete -c jmcp -n '__fish_seen_subcommand_from report' -l note -d 'Investigator note' -x

# snapshot flags
complete -c jmcp -n '__fish_seen_subcommand_from snapshot' -l label -d 'Snapshot label' -x
complete -c jmcp -n '__fish_seen_subcommand_from snapshot' -l errors -d 'Only error traces'
complete -c jmcp -n '__fish_seen_subcommand_from snapshot' -l since -d 'Lookback duration' -x
complete -c jmcp -n '__fish_seen_subcommand_from snapshot' -l depth -d 'Number of traces' -x
complete -c jmcp -n '__fish_seen_subcommand_from snapshot' -l dir -d 'Snapshot directory' -x

# completion
complete -c jmcp -n '__fish_seen_subcommand_from completion' -a 'bash zsh fish' -d 'Shell type'
`)
	return err
}
