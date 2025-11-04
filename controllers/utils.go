package controllers

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const (
	finalizerName = "aws.k8s.logmein.com"
)

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

func isTagPresent(tags []*ec2.Tag, searchedTag *ec2.Tag) bool {
	for _, tag := range tags {
		if tag.Key != nil && *tag.Key == *searchedTag.Key {
			return true
		}
	}
	return false
}

func convertMapToTags(tagMap map[string]string) []*ec2.Tag {
	var tags []*ec2.Tag
	for k, v := range tagMap {
		tags = append(tags, &ec2.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}
	return tags
}
