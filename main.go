package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/c-bata/go-prompt"
	"github.com/spf13/cobra"
	"github.com/winjeg/redis"
)

var Options = &struct {
	Auth string
}{}

type RedisOpiton struct {
	Ip       string
	Port     int
	ShowText string
	Slaves   []*RedisOpiton
}

func readRedisOpiton() []*RedisOpiton {
	scanner := bufio.NewScanner(os.Stdin)
	var redisClusters []*RedisOpiton
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		host := fields[0]
		port, err := strconv.Atoi(fields[1])
		if err != nil {
			log.Fatalln(err)
		}
		ips, err := net.LookupHost(host)
		if err != nil {
			log.Fatalln(err)
		}
		cluster := &RedisOpiton{
			Ip:   ips[0],
			Port: port,
		}
		if ips[0] != host {
			cluster.ShowText = fmt.Sprintf("%s:%d %s:%s", host, port, ips, "online")
		} else {
			cluster.ShowText = fmt.Sprintf("%s:%d:%s", ips[0], port, "online")
		}

		cluster.ShowText = fmt.Sprintf("%s:%d:%s", host, port, "online")
		cluster.Slaves = getSlaves(host, port)
		redisClusters = append(redisClusters, cluster)
	}
	return redisClusters
}

func getSlaves(host string, port int) []*RedisOpiton {
	var slaves []*RedisOpiton
	lines := commandText("replication", host, port)
	re := regexp.MustCompile("slave[0-9]*:ip=(.*),port=([0-9]+),state=([a-z]+).*")
	for _, line := range lines {
		if !strings.HasPrefix(line, "slave") {
			continue
		}
		matches := re.FindStringSubmatch(line)
		if len(matches) < 4 {
			continue
		}
		host := matches[1]
		port, err := strconv.Atoi(matches[2])
		if err != nil {
			log.Fatalln(err)
		}
		redisSlave := &RedisOpiton{
			Ip:   host,
			Port: port,
		}
		redisSlave.ShowText = fmt.Sprintf("%s:%d:%s", host, port, matches[3])
		redisSlave.Slaves = getSlaves(host, port)
		slaves = append(slaves, redisSlave)
	}
	return slaves
}

func spaces(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		s += " "
	}
	return s
}

func commandRecursive(command, prefix string, cluster []*RedisOpiton, call func(string, []string)) {
	if len(cluster) == 0 {
		fmt.Println("not found redis master/slave info")
		return
	}
	for i := range cluster {
		redis := cluster[i]
		fmt.Printf("%s%s\n", prefix, redis.ShowText)
		lines := commandText(command, redis.Ip, redis.Port)
		call(spaces(utf8.RuneCountInString(prefix)+2), lines)
		if len(redis.Slaves) > 0 {
			commandRecursive(command, spaces(utf8.RuneCountInString(prefix))+"├──", redis.Slaves, call)
		}
	}
}

func commandText(command, ip string, port int) []string {
	c := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", ip, port),
		Password: Options.Auth,
	})
	text, err := c.Do("info", command).String()
	if err != nil {
		log.Fatalln(err)
	}
	return strings.Split(text, "\n")
}

type command struct {
	redis []*RedisOpiton
}

func (cmd *command) memory(args []string) {
	call := callWrapper(args, true)
	commandRecursive("memory", "", cmd.redis, call)
}

func (cmd *command) replication(args []string) {
	call := callWrapper(args, false)
	commandRecursive("replication", "", cmd.redis, call)
}

func (cmd *command) server(args []string) {
	call := callWrapper(args, true)
	commandRecursive("server", "", cmd.redis, call)
}

func (cmd *command) clients(args []string) {
	call := callWrapper(args, true)
	commandRecursive("clients", "", cmd.redis, call)
}

func (cmd *command) stats(args []string) {
	call := callWrapper(args, true)
	commandRecursive("stats", "", cmd.redis, call)
}

func (cmd *command) persistence(args []string) {
	call := callWrapper(args, true)
	commandRecursive("persistence", "", cmd.redis, call)
}

func (cmd *command) cpu(args []string) {
	call := callWrapper(args, true)
	commandRecursive("cpu", "", cmd.redis, call)
}

func (cmd *command) cluster(args []string) {
	call := callWrapper(args, true)
	commandRecursive("cluster", "", cmd.redis, call)
}

func (cmd *command) keyspace(args []string) {
	call := callWrapper(args, true)
	commandRecursive("keyspace", "", cmd.redis, call)
}

func callWrapper(args []string, showAll bool) func(string, []string) {
	return func(prefix string, lines []string) {
		for i := range lines {
			if strings.HasPrefix(lines[i], "#") || len(lines[i]) == 0 {
				continue
			}
			if len(args) == 0 && showAll {
				fmt.Printf("%s+%s\n", prefix, lines[i])
				continue
			}
			for j := range args {
				if strings.HasPrefix(lines[i], args[j]) {
					fmt.Printf("%s+%s\n", prefix, lines[i])
				}
			}
		}
	}
}

func promptCompleter(d prompt.Document) []prompt.Suggest {
	s := []prompt.Suggest{
		{Text: "memory", Description: "memory <key1> [key2] ..."},
		{Text: "repliaction", Description: "replication <key1> [key2] ..."},
		{Text: "Server", Description: "server <key1> [key2] ..."},
		{Text: "Clients", Description: "clients <key1> [key2] ..."},
		{Text: "Persistence", Description: "persistence <key1> [key2] ..."},
		{Text: "Stats", Description: "stats <key1> [key2] ..."},
		{Text: "CPU", Description: "cpu <key1> [key2] ..."},
		{Text: "Cluster", Description: "cluster <key1> [key2] ..."},
		{Text: "Keyspace", Description: "keyspace <key1> [key2]"},
		{Text: "quit", Description: "quit the shell"},
		{Text: "exit", Description: "quit the shell"},
	}
	return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
}

func processLine(c *command, line string) {
	args := strings.Split(line, " ")
	if len(args) == 0 {
		return
	}
	cmd := args[0]
	switch cmd {
	case "memory":
		c.memory(args[1:])
	case "replication":
		c.replication(args[1:])
	case "server":
		c.server(args[1:])
	case "clients":
		c.clients(args[1:])
	case "persistence":
		c.persistence(args[1:])
	case "stats":
		c.stats(args[1:])
	case "cpu":
		c.cpu(args[1:])
	case "cluster":
		c.cluster(args[1:])
	case "keyspace":
		c.keyspace(args[1:])
	default:
		c.replication(args[1:])
	}
}

func cobraWapper(f func(args []string)) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		f(args)
	}
}

func main() {
	c := &command{}

	cmd := cobra.Command{Use: "redis-info"}
	cmd.PersistentFlags().StringVarP(&Options.Auth, "auth", "a", "", "redis auth")
	cmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		c.redis = readRedisOpiton()
	}
	cmd.Run = func(cmd *cobra.Command, args []string) {
		for {
			line := prompt.Input("> ", promptCompleter, prompt.OptionAddKeyBind(prompt.KeyBind{Key: prompt.ControlD, Fn: func(*prompt.Buffer) { os.Exit(0) }}))
			if line == "exit" || line == "quit" {
				os.Exit(0)
			}
			processLine(c, line)
		}
	}

	memory := &cobra.Command{Use: "memory <key1> [key2] ...", Run: cobraWapper(c.memory)}
	cmd.AddCommand(memory)
	replication := &cobra.Command{Use: "replication <key1> [key2] ...", Run: cobraWapper(c.replication)}
	cmd.AddCommand(replication)
	server := &cobra.Command{Use: "server <key1>", Run: cobraWapper(c.server)}
	cmd.AddCommand(server)
	clients := &cobra.Command{Use: "clients <key1>", Run: cobraWapper(c.clients)}
	cmd.AddCommand(clients)
	stats := &cobra.Command{Use: "stats <key1>", Run: cobraWapper(c.stats)}
	cmd.AddCommand(stats)
	cpu := &cobra.Command{Use: "cpu <key1>", Run: cobraWapper(c.cpu)}
	cmd.AddCommand(cpu)
	cluster := &cobra.Command{Use: "cluster <key1>", Run: cobraWapper(c.cluster)}
	cmd.AddCommand(cluster)
	keyspace := &cobra.Command{Use: "keyspace <key1>", Run: cobraWapper(c.keyspace)}
	cmd.AddCommand(keyspace)

	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
