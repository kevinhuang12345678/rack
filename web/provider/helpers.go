package provider

import (
	"fmt"
	"strings"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/crowdmob/goamz/dynamodb"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/goamz/goamz/cloudformation"
)

func appsTable(cluster string) *dynamodb.Table {
	pk := dynamodb.PrimaryKey{dynamodb.NewStringAttribute("name", ""), nil}
	table := DynamoDB.NewTable(fmt.Sprintf("%s-apps", cluster), pk)
	return table
}

func availabilityZones() ([]string, error) {
	res, err := EC2.DescribeAvailabilityZones(nil, nil)

	if err != nil {
		return nil, err
	}

	subnets := make([]string, len(res.AvailabilityZones))

	for i, zone := range res.AvailabilityZones {
		subnets[i] = zone.Name
	}

	return subnets, nil
}

func createStackFromTemplate(t, name string, tags map[string]string) error {
	params := &cloudformation.CreateStackParams{
		StackName:    name,
		TemplateBody: t,
	}

	for key, value := range tags {
		params.Tags = append(params.Tags, cloudformation.Tag{Key: key, Value: value})
	}

	_, err := CloudFormation.CreateStack(params)

	return err
}

func flattenTags(tags []cloudformation.Tag) map[string]string {
	f := make(map[string]string)

	for _, tag := range tags {
		f[tag.Key] = tag.Value
	}

	return f
}

func humanStatus(original string) string {
	switch original {
	case "CREATE_IN_PROGRESS":
		return "creating"
	case "CREATE_COMPLETE":
		return "running"
	case "DELETE_FAILED":
		return "running"
	case "DELETE_IN_PROGRESS":
		return "deleting"
	case "ROLLBACK_IN_PROGRESS":
		return "rollback"
	case "ROLLBACK_COMPLETE":
		return "failed"
	default:
		fmt.Printf("unknown status: %s\n", original)
		return "unknown"
	}
}

func divideSubnet(base string, num int) ([]string, error) {
	if num > 4 {
		return nil, fmt.Errorf("too many divisions")
	}

	div := make([]string, num)
	parts := strings.Split(base, ".")

	for i := 0; i < num; i++ {
		div[i] = fmt.Sprintf("%s.%s.%s.%d/27", parts[0], parts[1], parts[2], i*32)
	}

	return div, nil
}

func nextAvailableSubnet(vpc string) (string, error) {
	res, err := CloudFormation.DescribeStacks("", "")

	if err != nil {
		return "", err
	}

	available := make([]string, 254)

	for i := 1; i <= 254; i++ {
		available[i-1] = fmt.Sprintf("10.0.%d.0/24", i)
	}

	used := make([]string, 0)

	for _, stack := range res.Stacks {
		tags := stackTags(stack)
		if tags["type"] == "app" {
			used = append(used, tags["subnet"])
		}
	}

	for _, a := range available {
		found := false

		for _, u := range used {
			if a == u {
				found = true
				break
			}
		}

		if !found {
			return a, nil
		}
	}

	return "", fmt.Errorf("no available subnets")
}

func stackTags(stack cloudformation.Stack) map[string]string {
	tags := make(map[string]string)

	for _, tag := range stack.Tags {
		tags[tag.Key] = tag.Value
	}

	return tags
}

func stackOutputs(stack string) (map[string]string, error) {
	res, err := CloudFormation.DescribeStacks(stack, "")

	if err != nil {
		return nil, err
	}

	if len(res.Stacks) != 1 {
		return nil, fmt.Errorf("could not fetch stack %s", stack)
	}

	outputs := make(map[string]string)

	for _, output := range res.Stacks[0].Outputs {
		outputs[output.OutputKey] = output.OutputValue
	}

	return outputs, nil
}

func stackOutputList(stack, prefix string) ([]string, error) {
	outputs, err := stackOutputs(stack)

	if err != nil {
		return nil, err
	}

	values := make([]string, 0)

	for key, value := range outputs {
		if strings.HasPrefix(key, prefix) {
			values = append(values, value)
		}
	}

	return values, nil
}

func upperName(name string) string {
	return strings.ToUpper(name[0:1]) + name[1:]
}