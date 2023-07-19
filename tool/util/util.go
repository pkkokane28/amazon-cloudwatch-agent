// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package util

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	sysruntime "runtime"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"

	configaws "github.com/aws/private-amazon-cloudwatch-agent-staging/cfg/aws"
	"github.com/aws/private-amazon-cloudwatch-agent-staging/tool/data/interfaze"
	"github.com/aws/private-amazon-cloudwatch-agent-staging/tool/runtime"
	"github.com/aws/private-amazon-cloudwatch-agent-staging/tool/stdin"
)

const (
	configJsonFileName = "config.json"

	OsTypeLinux   = "linux"
	OsTypeWindows = "windows"
	OsTypeDarwin  = "darwin"

	MapKeyMetricsCollectionInterval = "metrics_collection_interval"
	MapKeyInstances                 = "resources"
	MapKeyMeasurement               = "measurement"
)

func CurOS() string {
	return sysruntime.GOOS
}

func CurPath() string {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	return path.Dir(ex)
}

func ConfigFilePath() string {
	return filepath.Join(CurPath(), configJsonFileName)
}

func PermissionCheck() {
	filePath := ConfigFilePath()
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		fmt.Printf("Make sure that you have write permission to %s\n", filePath)
		os.Exit(1)
	}
	defer f.Close()
	return
}

func ReadConfigFromJsonFile() string {
	filePath := ConfigFilePath()
	byteArray, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error in reading config from file %s: %v\n", filePath, err)
		os.Exit(1)
	}
	return string(byteArray)
}

func SerializeResultMapToJsonByteArray(resultMap map[string]interface{}) []byte {
	resultByteArray, err := json.MarshalIndent(resultMap, "", "\t")
	if err != nil {
		fmt.Printf("Result map to byte array json marshal error: %v\n", err)
		os.Exit(1)
	}
	return resultByteArray
}

func SaveResultByteArrayToJsonFile(resultByteArray []byte) string {
	filePath := ConfigFilePath()
	err := os.WriteFile(filePath, resultByteArray, 0755)
	if err != nil {
		fmt.Printf("Error in writing file to %s: %v\nMake sure that you have write permission to %s.", filePath, err, filePath)
		os.Exit(1)
	}
	fmt.Printf("Saved config file to %s successfully.\n", filePath)
	return filePath
}

func SDKRegion() (region string) {
	ses, err := session.NewSession()

	if err != nil {
		return
	}
	if ses.Config != nil && ses.Config.Region != nil {
		region = *ses.Config.Region
	}
	return region
}

func SDKRegionWithProfile(profile string) (region string) {
	ses, err := session.NewSessionWithOptions(session.Options{Profile: profile, SharedConfigState: session.SharedConfigEnable})

	if err != nil {
		return
	}
	if ses.Config != nil && ses.Config.Region != nil {
		region = *ses.Config.Region
	}
	return region
}

func SDKCredentials() (accessKey, secretKey string, creds *credentials.Credentials) {
	ses, err := session.NewSession()
	if err != nil {
		return
	}
	if ses.Config != nil && ses.Config.Credentials != nil {
		if credsValue, err := ses.Config.Credentials.Get(); err == nil {
			accessKey = credsValue.AccessKeyID
			secretKey = credsValue.SecretAccessKey
			creds = ses.Config.Credentials
		}
	}
	return
}

func DefaultEC2Region() (region string) {
	fmt.Println("Trying to fetch the default region based on ec2 metadata...")
	// imds does not need to retry here since this is config wizard
	// by the time user can run the wizard imds should be up
	ses, err := session.NewSession(&aws.Config{
		HTTPClient: &http.Client{Timeout: 1 * time.Second},
		MaxRetries: aws.Int(0),
		LogLevel:   configaws.SDKLogLevel(),
		Logger:     configaws.SDKLogger{},
	})
	if err != nil {
		return
	}
	md := ec2metadata.New(ses)
	if info, err := md.Region(); err == nil {
		region = info
	} else {
		fmt.Println("Could not get region from ec2 metadata...")
	}
	return
}

func AddToMap(ctx *runtime.Context, resultMap map[string]interface{}, obj interfaze.ConvertibleToMap) {
	key, value := obj.ToMap(ctx)
	if key != "" && value != nil {
		resultMap[key] = value
	}
}

func Yes(question string) bool {
	answer := Choice(question, 1, []string{"yes", "no"})
	return answer == "yes"
}

func No(question string) bool {
	answer := Choice(question, 2, []string{"yes", "no"})
	return answer == "yes"
}

func AskWithDefault(question, defaultValue string) string {
	for {
		var answer string
		fmt.Printf("%s\ndefault choice: [%s]\n\r", question, defaultValue)

		stdin.Scanln(&answer)

		if answer == "" {
			return defaultValue
		}
		return answer
	}
}

func Ask(question string) string {
	return Choice(question, 0, nil)
}

// defaultOption value starts from 1
func Choice(question string, defaultOption int, validValues []string) string {
	for {
		var answer string
		options := ""
		if validValues != nil {
			for i := range validValues {
				options = fmt.Sprintf("%s%s. %s\n", options, strconv.Itoa(i+1), validValues[i])
			}
			fmt.Printf("%s\n%sdefault choice: [%d]:\n\r", question, options, defaultOption)
		} else {
			fmt.Printf("%s\n\r", question)
		}

		stdin.Scanln(&answer)

		if validValues == nil {
			return answer
		}

		var option int
		var err error
		if answer == "" {
			option = defaultOption
		} else {
			option, err = strconv.Atoi(answer)
		}
		if err == nil && option > 0 && option <= len(validValues) {
			return validValues[option-1]
		}
		fmt.Printf("The value %s is not valid to this question.\nPlease retry to answer:\n", answer)
	}
}

func EnterToExit() {
	fmt.Println("Please press Enter to exit...")
	stdin.Scanln()
}
