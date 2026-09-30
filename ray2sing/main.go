package main

import (
	_ "embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/HZ-PRE/ray2sing/ray2sing"
	"github.com/sagernet/sing-box/experimental/libbox"
	_ "github.com/sagernet/sing-box/include"
)

var examples = map[string][]string{
	"vless": {"vless://00000000-0000-4000-8000-000000000001@example.com:443?security=tls&type=ws#VLESS"},
}

func main() {
	// Replace "path/to/your/config/file" with the actual path to your config file
	var configs string
	if len(os.Args) > 1 {
		if len(examples[os.Args[1]]) != 0 {
			configs = strings.Join(examples[os.Args[1]], "\n")
			fmt.Printf("%s\n", configs)
		} else {
			configs = strings.Join(os.Args[1:], "\n")
		}
	} else {
		configs = read()
	}
	clash_conf, err := ray2sing.Ray2Singbox(libbox.BaseContext(nil), configs, false)

	if err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	fmt.Printf("Parsed config: \n%+v\n", string(clash_conf))
	fmt.Printf("==============\n===========\n=============")

}

func read() string {
	url := "https://raw.githubusercontent.com/ImMyron/V2ray/main/V2ray.txt"

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching URL content:", err)
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return ""
	}

	fmt.Println("URL Content:")
	fmt.Println(string(body))
	return string(body)
}
