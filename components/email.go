package components

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

var client *sesv2.Client
var sendFrom string = os.Getenv("EMAIL_FROM")

func init() {
	if client != nil {
		return
	}
	var err error
	var cfg aws.Config

	if cfg, err = config.LoadDefaultConfig(context.TODO(), config.WithRegion(os.Getenv("AWS_REGION"))); err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	client = sesv2.NewFromConfig(cfg)
}

func SendEmailRaw(to string, subject string, body string) error {
	resp, err := client.SendEmail(context.TODO(), &sesv2.SendEmailInput{
		FromEmailAddress: &sendFrom,
		Destination: &types.Destination{
			ToAddresses: []string{to},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Body: &types.Body{
					Html: &types.Content{
						Charset: aws.String("UTF-8"),
						Data:    &body,
					},
				},
				Subject: &types.Content{
					Charset: aws.String("UTF-8"),
					Data:    &subject,
				},
			},
		},
	})
	if err == nil {
		respJson, _ := json.MarshalIndent(resp.MessageId, "", "  ")
		log.Println("Send email response:", string(respJson))
	}
	return err
}

func SendEmailTemplate(to string, templateName string, templateData string) error {
	resp, err := client.SendEmail(context.TODO(), &sesv2.SendEmailInput{
		FromEmailAddress: &sendFrom,
		Destination: &types.Destination{
			ToAddresses: []string{to},
		},
		Content: &types.EmailContent{
			Template: &types.Template{
				TemplateName: &templateName,
				TemplateData: &templateData,
			},
		},
	})
	if err == nil {
		respJson, _ := json.MarshalIndent(resp.MessageId, "", "  ")
		log.Println("Send email template response:", string(respJson))
	}
	return err
}

func CreateTemplate(name string, subject string, body string) error {
	_, err := client.CreateEmailTemplate(context.TODO(), &sesv2.CreateEmailTemplateInput{
		TemplateName: &name,
		TemplateContent: &types.EmailTemplateContent{
			Subject: &subject,
			Html:    &body,
		},
	})
	if err == nil {
		log.Println("Create template response")
	}
	return err
}

func ListTemplates() error {
	resp, err := client.ListEmailTemplates(context.TODO(), &sesv2.ListEmailTemplatesInput{
		PageSize: aws.Int32(10),
	})
	if err == nil {
		respJson, _ := json.MarshalIndent(resp.TemplatesMetadata, "", "  ")
		log.Println("List templates response:", string(respJson))
	}
	return err
}

func UpdateTemplate(name string, subject string, body string) error {
	_, err := client.UpdateEmailTemplate(context.TODO(), &sesv2.UpdateEmailTemplateInput{
		TemplateName: &name,
		TemplateContent: &types.EmailTemplateContent{
			Subject: &subject,
			Html:    &body,
		},
	})
	if err == nil {
		log.Println("Update template response")
	}
	return err
}

func DeleteTemplate(name string) error {
	_, err := client.DeleteEmailTemplate(context.TODO(), &sesv2.DeleteEmailTemplateInput{
		TemplateName: &name,
	})
	if err == nil {
		log.Println("Delete template response")
	}
	return err
}
