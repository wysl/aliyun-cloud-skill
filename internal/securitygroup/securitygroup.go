package securitygroup

import (
	"aliyun-cloud-skill/internal/aliyuncli"
	"encoding/json"
	"fmt"
	"strings"
)

// SecurityGroup represents a security group
type SecurityGroup struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	VpcID         string   `json:"vpcId"`
	Type          string   `json:"type"`
	InstanceIDs   []string `json:"instanceIds,omitempty"`
	InstanceNames []string `json:"instanceNames,omitempty"`
}

// Rule represents a security group rule
type Rule struct {
	Direction     string `json:"direction"`     // ingress/egress
	PortRange     string `json:"portRange"`
	Protocol      string `json:"protocol"`
	SourceCidr    string `json:"sourceCidr,omitempty"`
	DestCidr      string `json:"destCidr,omitempty"`
	SourceGroupID string `json:"sourceGroupId,omitempty"`
	Policy        string `json:"policy"` // Accept/Drop
	Description   string `json:"description,omitempty"`
}

// List returns all security groups in a region
func List(region string, env map[string]string) ([]SecurityGroup, error) {
	args := []string{"ecs", "DescribeSecurityGroups", "--RegionId", region}
	var data struct {
		SecurityGroups struct {
			SecurityGroup []struct {
				SecurityGroupId   string `json:"SecurityGroupId"`
				SecurityGroupName string `json:"SecurityGroupName"`
				VpcId             string `json:"VpcId"`
				SecurityGroupType string `json:"SecurityGroupType"`
			} `json:"SecurityGroup"`
		} `json:"SecurityGroups"`
	}
	if err := aliyuncli.RunJSON(args, env, &data); err != nil {
		return nil, err
	}

	res := make([]SecurityGroup, 0, len(data.SecurityGroups.SecurityGroup))
	for _, sg := range data.SecurityGroups.SecurityGroup {
		res = append(res, SecurityGroup{
			ID:    sg.SecurityGroupId,
			Name:  sg.SecurityGroupName,
			VpcID: sg.VpcId,
			Type:  sg.SecurityGroupType,
		})
	}
	return res, nil
}

// ListWithInstances returns security groups with bound instance info
func ListWithInstances(region string, env map[string]string) ([]SecurityGroup, error) {
	// First get all security groups
	sgs, err := List(region, env)
	if err != nil {
		return nil, err
	}

	// Build a map of security group ID to instance IDs
	sgInstanceMap := make(map[string][]string)
	sgInstanceNameMap := make(map[string][]string)

	// Query all instances and their security groups
	args := []string{"ecs", "DescribeInstances", "--RegionId", region}
	var data struct {
		Instances struct {
			Instance []struct {
				InstanceId   string `json:"InstanceId"`
				InstanceName string `json:"InstanceName"`
				SecurityGroupIds struct {
					SecurityGroupId []string `json:"SecurityGroupId"`
				} `json:"SecurityGroupIds"`
			} `json:"Instance"`
		} `json:"Instances"`
	}
	if err := aliyuncli.RunJSON(args, env, &data); err != nil {
		return nil, err
	}

	// Build the mapping
	for _, inst := range data.Instances.Instance {
		instName := inst.InstanceName
		if instName == "" {
			instName = inst.InstanceId
		}
		for _, sgID := range inst.SecurityGroupIds.SecurityGroupId {
			sgInstanceMap[sgID] = append(sgInstanceMap[sgID], inst.InstanceId)
			sgInstanceNameMap[sgID] = append(sgInstanceNameMap[sgID], instName)
		}
	}

	// Attach instance info to security groups
	for i := range sgs {
		sgs[i].InstanceIDs = sgInstanceMap[sgs[i].ID]
		sgs[i].InstanceNames = sgInstanceNameMap[sgs[i].ID]
	}

	return sgs, nil
}

// GetRules returns the rules of a security group
func GetRules(region, securityGroupID string, env map[string]string) ([]Rule, error) {
	args := []string{"ecs", "DescribeSecurityGroupAttribute", "--RegionId", region, "--SecurityGroupId", securityGroupID}
	var data struct {
		Permissions struct {
			Permission []struct {
				Direction       string `json:"Direction"`
				PortRange       string `json:"PortRange"`
				IpProtocol      string `json:"IpProtocol"`
				SourceCidrIp    string `json:"SourceCidrIp"`
				DestCidrIp      string `json:"DestCidrIp"`
				SourceGroupId   string `json:"SourceGroupId"`
				Policy          string `json:"Policy"`
				Description     string `json:"Description"`
			} `json:"Permission"`
		} `json:"Permissions"`
	}
	if err := aliyuncli.RunJSON(args, env, &data); err != nil {
		return nil, err
	}

	rules := make([]Rule, 0, len(data.Permissions.Permission))
	for _, p := range data.Permissions.Permission {
		rule := Rule{
			Direction:     p.Direction,
			PortRange:     p.PortRange,
			Protocol:      p.IpProtocol,
			SourceCidr:    p.SourceCidrIp,
			DestCidr:      p.DestCidrIp,
			SourceGroupID: p.SourceGroupId,
			Policy:        p.Policy,
			Description:   p.Description,
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

// FormatList formats security group list with instance bindings
func FormatList(items []SecurityGroup, output string) string {
	if output == "json" {
		b, _ := json.MarshalIndent(items, "", "  ")
		return string(b)
	}
	if len(items) == 0 {
		return "未找到安全组"
	}
	lines := []string{fmt.Sprintf("安全组数量: %d", len(items)), ""}
	for _, sg := range items {
		name := sg.Name
		if name == "" {
			name = sg.ID
		}
		lines = append(lines, fmt.Sprintf("- %s (%s)", name, sg.ID))
		if len(sg.InstanceNames) > 0 {
			lines = append(lines, fmt.Sprintf("  绑定实例: %s", strings.Join(sg.InstanceNames, ", ")))
		} else {
			lines = append(lines, "  绑定实例: 无")
		}
	}
	return strings.Join(lines, "\n")
}

// FormatRules formats security group rules
func FormatRules(sgID string, rules []Rule, output string) string {
	if output == "json" {
		data := map[string]any{
			"securityGroupId": sgID,
			"rules":           rules,
		}
		b, _ := json.MarshalIndent(data, "", "  ")
		return string(b)
	}
	if len(rules) == 0 {
		return fmt.Sprintf("安全组 %s 无规则", sgID)
	}

	// Separate ingress and egress rules
	ingressRules := []Rule{}
	egressRules := []Rule{}
	for _, r := range rules {
		if r.Direction == "ingress" {
			ingressRules = append(ingressRules, r)
		} else {
			egressRules = append(egressRules, r)
		}
	}

	lines := []string{fmt.Sprintf("安全组: %s", sgID), ""}

	if len(ingressRules) > 0 {
		lines = append(lines, "=== 入方向规则 ===")
		lines = append(lines, "")
		lines = append(lines, "协议 | 端口范围 | 源地址 | 策略 | 描述")
		lines = append(lines, strings.Repeat("-", 60))
		for _, r := range ingressRules {
			source := r.SourceCidr
			if source == "" {
				source = r.SourceGroupID
			}
			if source == "" {
				source = "N/A"
			}
			desc := r.Description
			if desc == "" {
				desc = "-"
			}
			lines = append(lines, fmt.Sprintf("%s | %s | %s | %s | %s", r.Protocol, r.PortRange, source, r.Policy, desc))
		}
		lines = append(lines, "")
	}

	if len(egressRules) > 0 {
		lines = append(lines, "=== 出方向规则 ===")
		lines = append(lines, "")
		lines = append(lines, "协议 | 端口范围 | 目标地址 | 策略 | 描述")
		lines = append(lines, strings.Repeat("-", 60))
		for _, r := range egressRules {
			dest := r.DestCidr
			if dest == "" {
				dest = "N/A"
			}
			desc := r.Description
			if desc == "" {
				desc = "-"
			}
			lines = append(lines, fmt.Sprintf("%s | %s | %s | %s | %s", r.Protocol, r.PortRange, dest, r.Policy, desc))
		}
	}

	return strings.Join(lines, "\n")
}

// AddIngressRule adds an ingress rule to a security group
func AddIngressRule(region, securityGroupID, protocol, portRange, sourceCidr, policy, description string, priority int, env map[string]string) error {
	args := []string{
		"ecs", "AuthorizeSecurityGroup",
		"--RegionId", region,
		"--SecurityGroupId", securityGroupID,
		"--IpProtocol", protocol,
		"--PortRange", portRange,
		"--SourceCidrIp", sourceCidr,
		"--Policy", policy,
		"--Priority", fmt.Sprintf("%d", priority),
	}
	if description != "" {
		args = append(args, "--Description", description)
	}
	_, err := aliyuncli.RunRaw(args, env)
	return err
}

// AddEgressRule adds an egress rule to a security group
func AddEgressRule(region, securityGroupID, protocol, portRange, destCidr, policy, description string, priority int, env map[string]string) error {
	args := []string{
		"ecs", "AuthorizeSecurityGroupEgress",
		"--RegionId", region,
		"--SecurityGroupId", securityGroupID,
		"--IpProtocol", protocol,
		"--PortRange", portRange,
		"--DestCidrIp", destCidr,
		"--Policy", policy,
		"--Priority", fmt.Sprintf("%d", priority),
	}
	if description != "" {
		args = append(args, "--Description", description)
	}
	_, err := aliyuncli.RunRaw(args, env)
	return err
}

// JoinSecurityGroup adds an ECS instance to a security group
func JoinSecurityGroup(region, securityGroupID, instanceID string, env map[string]string) error {
	args := []string{
		"ecs", "JoinSecurityGroup",
		"--RegionId", region,
		"--SecurityGroupId", securityGroupID,
		"--InstanceId", instanceID,
	}
	_, err := aliyuncli.RunRaw(args, env)
	return err
}

// LeaveSecurityGroup removes an ECS instance from a security group
func LeaveSecurityGroup(region, securityGroupID, instanceID string, env map[string]string) error {
	args := []string{
		"ecs", "LeaveSecurityGroup",
		"--RegionId", region,
		"--SecurityGroupId", securityGroupID,
		"--InstanceId", instanceID,
	}
	_, err := aliyuncli.RunRaw(args, env)
	return err
}