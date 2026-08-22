package awsx

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/pricing"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

const (
	TagKey   = "ManagedBy"
	TagValue = "sharingan"

	// The Pricing API answers only through us-east-1, whatever region is being priced.
	pricingRegion = "us-east-1"
)

type Clients struct {
	EC2     *ec2.Client
	SSM     *ssm.Client
	STS     *sts.Client
	Pricing *pricing.Client
	Region  string
	Profile string
	Account string
}

func New(ctx context.Context, profile, region string) (*Clients, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithSharedConfigProfile(profile),
		awsconfig.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("loading aws config for profile %s in %s: %w", profile, region, err)
	}
	pricingCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithSharedConfigProfile(profile),
		awsconfig.WithRegion(pricingRegion),
	)
	if err != nil {
		return nil, fmt.Errorf("loading aws config for profile %s in %s: %w", profile, pricingRegion, err)
	}

	stsClient := sts.NewFromConfig(cfg)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, fmt.Errorf("sts:GetCallerIdentity for profile %s: %w", profile, err)
	}

	return &Clients{
		EC2:     ec2.NewFromConfig(cfg),
		SSM:     ssm.NewFromConfig(cfg),
		STS:     stsClient,
		Pricing: pricing.NewFromConfig(pricingCfg),
		Region:  region,
		Profile: profile,
		Account: aws.ToString(identity.Account),
	}, nil
}

func ManagedFilter() []ec2types.Filter {
	return []ec2types.Filter{{
		Name:   aws.String("tag:" + TagKey),
		Values: []string{TagValue},
	}}
}

func NameFilter(name string) ec2types.Filter {
	return ec2types.Filter{
		Name:   aws.String("tag:Name"),
		Values: []string{name},
	}
}

func TagSpecs(rt ec2types.ResourceType, name string) ec2types.TagSpecification {
	tags := []ec2types.Tag{{Key: aws.String(TagKey), Value: aws.String(TagValue)}}
	if name != "" {
		tags = append(tags, ec2types.Tag{Key: aws.String("Name"), Value: aws.String(name)})
	}
	return ec2types.TagSpecification{ResourceType: rt, Tags: tags}
}
