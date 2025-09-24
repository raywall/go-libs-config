package main

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/raywall/go-libs-config/builder"
)

var ctx = context.Background()

func main() {
	cfg, _ := config.LoadDefaultConfig(context.TODO())
	ssmClient := ssm.NewFromConfig(cfg)

	svc := builder.New(ssmClient)
	content, err := svc.BuildJsonFromPrefix(ctx, "/teste/app/schema", true)
	if err != nil {
		panic(err)
	}
	rules, err := svc.BuildYamlFromPrefix(ctx, "/teste/app/rules", true)
	if err != nil {
		panic(err)
	}

	// schema
	log.Println("Schema completo:\n", string(content))

	// rules
	log.Println("Rules completo:\n", string(rules))
}
