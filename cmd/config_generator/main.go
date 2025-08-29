package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"

	"github.com/ChiaYuChang/weathercock/internal/cfggen"
)

func main() {
	// 設置 Viper 從環境變數讀取
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // 處理嵌套結構的環境變數

	// 讀取 .env 檔案到全局 viper 實例，作為 cfggen 的 sourceViper
	viper.SetConfigFile(".env")
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Warning: Error reading .env file, using environment variables only: %v\n", err)
	}

	serviceName := flag.String("service", "", "Specify the service config to generate (e.g., scraper, logger, keyword_extractor)")
	all := flag.Bool("all", false, "Generate configs for all services")
	flag.Parse()

	// 確保 configs 目錄存在
	outputDir := "configs"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("Error creating configs directory: %v\n", err)
		os.Exit(1)
	}

	if *all {
		fmt.Println("Generating configs for all services...")
		generateAllConfigs(outputDir)
	} else if *serviceName != "" {
		fmt.Printf("Generating config for service: %s\n", *serviceName)
		generateServiceConfig(*serviceName, outputDir)
	} else {
		fmt.Println("Please specify a service to generate config for using --service <service_name> or --all to generate all configs.")
		flag.PrintDefaults()
		os.Exit(1)
	}

	fmt.Println("Config generation complete.")
}

func generateServiceConfig(service string, outputDir string) {
	// Pass the global viper instance (which has read .env) as the source
	cfgGen := cfggen.NewCfgGen(viper.GetViper())
	outputFile := filepath.Join(outputDir, fmt.Sprintf("%s.json", service))

	switch service {
	case "scraper":
		cfgGen.AddLoggerConfig()
		cfgGen.AddPostgresConfig()
		cfgGen.AddNATSConfig()
		cfgGen.AddValkeyConfig()
		cfgGen.AddScraperConfig()
	case "logger":
		cfgGen.AddPostgresConfig()
		cfgGen.AddNATSConfig()
		cfgGen.AddLoggerConfig()
	case "keyword_extractor":
		cfgGen.AddPostgresConfig()
		cfgGen.AddNATSConfig()
		cfgGen.AddValkeyConfig()
		cfgGen.AddLLMConfig()
		cfgGen.AddKeywordExtractorConfig()
	case "api": // 假設 API 服務也需要配置
		cfgGen.AddPostgresConfig()
		cfgGen.AddNATSConfig()
		cfgGen.AddValkeyConfig()
		cfgGen.AddLLMConfig()
		cfgGen.AddTemplatesConfig()
	case "migrate": // 假設 migrate 服務也需要配置
		cfgGen.AddPostgresConfig()
		cfgGen.AddMigrationConfig()
	case "party_press_release_scraper": // 假設 party_press_release_scraper 服務也需要配置
		cfgGen.AddPostgresConfig()
		cfgGen.AddNATSConfig()
		cfgGen.AddValkeyConfig()
		cfgGen.AddScraperConfig() // 假設它使用類似 Scraper 的配置
	case "testdata": // 假設 testdata 服務也需要配置
		cfgGen.AddPostgresConfig()
		cfgGen.AddNATSConfig()
		cfgGen.AddValkeyConfig()
		cfgGen.AddLLMConfig()
	default:
		fmt.Printf("Unknown service: %s. No config generated.\n", service)
		return
	}

	file, err := os.Create(outputFile)
	if err != nil {
		fmt.Printf("Error creating output file %s: %v\n", outputFile, err)
		return
	}
	defer file.Close()

	if err := cfgGen.WriteTo(file, "json"); err != nil {
		fmt.Printf("Error writing config for service %s to %s: %v\n", service, outputFile, err)
		return
	}
	fmt.Printf("Config for service '%s' successfully generated to %s\n", service, outputFile)
}

func generateAllConfigs(outputDir string) {
	services := []string{
		"scraper",
		"logger",
		"keyword_extractor",
		"api",
		"migrate",
		"party_press_release_scraper",
		"testdata",
	}

	for _, service := range services {
		generateServiceConfig(service, outputDir)
	}
}
