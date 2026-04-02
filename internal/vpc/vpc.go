package vpc

import (
	"aliyun-cloud-monitor/internal/aliyuncli"
	"encoding/json"
	"fmt"
	"strings"
)

// VPC structures
type VPC struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	CidrBlock   string `json:"cidrBlock"`
	VRouterId   string `json:"vRouterId"`
	IsDefault   bool   `json:"isDefault"`
	Description string `json:"description"`
}

type VSwitch struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	CidrBlock   string `json:"cidrBlock"`
	VpcId       string `json:"vpcId"`
	ZoneId      string `json:"zoneId"`
	Description string `json:"description"`
}

type RouteTable struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	VpcId       string `json:"vpcId"`
	VRouterId   string `json:"vRouterId"`
	Description string `json:"description"`
}

type RouteEntry struct {
	ID              string `json:"id"`
	DestinationCidr string `json:"destinationCidr"`
	NextHopType     string `json:"nextHopType"`
	NextHopId       string `json:"nextHopId"`
	Type            string `json:"type"`
	RouteTableId    string `json:"routeTableId"`
}

type VPCDetail struct {
	VPC          VPC          `json:"vpc"`
	VSwitches    []VSwitch    `json:"vswitches"`
	RouteTables  []RouteTable `json:"routeTables"`
	RouteEntries []RouteEntry `json:"routeEntries"`
}

// ListVPCs lists all VPCs in a region
func ListVPCs(region string, env map[string]string) ([]VPC, error) {
	args := []string{"vpc", "DescribeVpcs", "--RegionId", region}
	var data struct {
		Vpcs struct {
			Vpc []struct {
				VpcId       string `json:"VpcId"`
				VpcName     string `json:"VpcName"`
				Status      string `json:"Status"`
				CidrBlock   string `json:"CidrBlock"`
				VRouterId   string `json:"VRouterId"`
				IsDefault   bool   `json:"IsDefault"`
				Description string `json:"Description"`
			} `json:"Vpc"`
		} `json:"Vpcs"`
	}
	if err := aliyuncli.RunJSON(args, env, &data); err != nil {
		return nil, err
	}
	res := make([]VPC, 0, len(data.Vpcs.Vpc))
	for _, it := range data.Vpcs.Vpc {
		res = append(res, VPC{
			ID:          it.VpcId,
			Name:        it.VpcName,
			Status:      it.Status,
			CidrBlock:   it.CidrBlock,
			VRouterId:   it.VRouterId,
			IsDefault:   it.IsDefault,
			Description: it.Description,
		})
	}
	return res, nil
}

// GetVPCAttribute gets detailed info about a VPC
func GetVPCAttribute(region, vpcId string, env map[string]string) (VPC, error) {
	args := []string{"vpc", "DescribeVpcAttribute", "--RegionId", region, "--VpcId", vpcId}
	var data struct {
		VpcId       string `json:"VpcId"`
		VpcName     string `json:"VpcName"`
		Status      string `json:"Status"`
		CidrBlock   string `json:"CidrBlock"`
		VRouterId   string `json:"VRouterId"`
		IsDefault   bool   `json:"IsDefault"`
		Description string `json:"Description"`
	}
	if err := aliyuncli.RunJSON(args, env, &data); err != nil {
		return VPC{}, err
	}
	return VPC{
		ID:          data.VpcId,
		Name:        data.VpcName,
		Status:      data.Status,
		CidrBlock:   data.CidrBlock,
		VRouterId:   data.VRouterId,
		IsDefault:   data.IsDefault,
		Description: data.Description,
	}, nil
}

// ListVSwitches lists all VSwitches in a region or VPC
func ListVSwitches(region, vpcId string, env map[string]string) ([]VSwitch, error) {
	args := []string{"vpc", "DescribeVSwitches", "--RegionId", region}
	if vpcId != "" {
		args = append(args, "--VpcId", vpcId)
	}
	var data struct {
		VSwitches struct {
			VSwitch []struct {
				VSwitchId   string `json:"VSwitchId"`
				VSwitchName string `json:"VSwitchName"`
				Status      string `json:"Status"`
				CidrBlock   string `json:"CidrBlock"`
				VpcId       string `json:"VpcId"`
				ZoneId      string `json:"ZoneId"`
				Description string `json:"Description"`
			} `json:"VSwitch"`
		} `json:"VSwitches"`
	}
	if err := aliyuncli.RunJSON(args, env, &data); err != nil {
		return nil, err
	}
	res := make([]VSwitch, 0, len(data.VSwitches.VSwitch))
	for _, it := range data.VSwitches.VSwitch {
		res = append(res, VSwitch{
			ID:          it.VSwitchId,
			Name:        it.VSwitchName,
			Status:      it.Status,
			CidrBlock:   it.CidrBlock,
			VpcId:       it.VpcId,
			ZoneId:      it.ZoneId,
			Description: it.Description,
		})
	}
	return res, nil
}

// ListRouteTables lists all route tables in a region or VPC
func ListRouteTables(region, vpcId string, env map[string]string) ([]RouteTable, error) {
	// If vpcId is not specified, get all VPCs first and query each VPC's route tables
	if vpcId == "" {
		vpcs, err := ListVPCs(region, env)
		if err != nil {
			return nil, err
		}
		allTables := []RouteTable{}
		for _, v := range vpcs {
			tables, err := listRouteTablesByVpc(region, v.ID, env)
			if err != nil {
				continue
			}
			allTables = append(allTables, tables...)
		}
		return allTables, nil
	}
	return listRouteTablesByVpc(region, vpcId, env)
}

// listRouteTablesByVpc lists route tables for a specific VPC
func listRouteTablesByVpc(region, vpcId string, env map[string]string) ([]RouteTable, error) {
	// First get VPC to obtain VRouterId
	vpcInfo, err := GetVPCAttribute(region, vpcId, env)
	if err != nil {
		return nil, err
	}
	
	args := []string{"vpc", "DescribeRouteTables", "--RegionId", region, "--RouterId", vpcInfo.VRouterId, "--RouterType", "VRouter"}
	var data struct {
		RouteTables struct {
			RouteTable []struct {
				RouteTableId   string `json:"RouteTableId"`
				RouteTableName string `json:"RouteTableName"`
				RouteTableType string `json:"RouteTableType"`
				VpcId          string `json:"VpcId"`
				VRouterId      string `json:"VRouterId"`
				Description    string `json:"Description"`
			} `json:"RouteTable"`
		} `json:"RouteTables"`
	}
	if err := aliyuncli.RunJSON(args, env, &data); err != nil {
		return nil, err
	}
	res := make([]RouteTable, 0, len(data.RouteTables.RouteTable))
	for _, it := range data.RouteTables.RouteTable {
		res = append(res, RouteTable{
			ID:          it.RouteTableId,
			Name:        it.RouteTableName,
			Type:        it.RouteTableType,
			VpcId:       it.VpcId,
			VRouterId:   it.VRouterId,
			Description: it.Description,
		})
	}
	return res, nil
}

// ListRouteEntries lists route entries for a route table
func ListRouteEntries(region, routeTableId string, env map[string]string) ([]RouteEntry, error) {
	args := []string{"vpc", "DescribeRouteEntryList", "--RegionId", region, "--RouteTableId", routeTableId}
	var data struct {
		RouteEntrys struct {
			RouteEntry []struct {
				RouteEntryId         string `json:"RouteEntryId"`
				DestinationCidrBlock string `json:"DestinationCidrBlock"`
				NextHops             struct {
					NextHop []struct {
						NextHopId   string `json:"NextHopId"`
						NextHopType string `json:"NextHopType"`
					} `json:"NextHop"`
				} `json:"NextHops"`
				RouteEntryType string `json:"Type"`
				RouteTableId   string `json:"RouteTableId"`
			} `json:"RouteEntry"`
		} `json:"RouteEntrys"`
	}
	if err := aliyuncli.RunJSON(args, env, &data); err != nil {
		return nil, err
	}
	res := make([]RouteEntry, 0, len(data.RouteEntrys.RouteEntry))
	for _, it := range data.RouteEntrys.RouteEntry {
		// Get first next hop info
		nextHopType := ""
		nextHopId := ""
		if len(it.NextHops.NextHop) > 0 {
			nextHopType = it.NextHops.NextHop[0].NextHopType
			nextHopId = it.NextHops.NextHop[0].NextHopId
		}
		res = append(res, RouteEntry{
			ID:              it.RouteEntryId,
			DestinationCidr: it.DestinationCidrBlock,
			NextHopType:     nextHopType,
			NextHopId:       nextHopId,
			Type:            it.RouteEntryType,
			RouteTableId:    it.RouteTableId,
		})
	}
	return res, nil
}

// GetVPCDetail gets full VPC detail including vswitches, route tables, and route entries
func GetVPCDetail(region, vpcId string, env map[string]string) (VPCDetail, error) {
	// Get VPC info
	vpc, err := GetVPCAttribute(region, vpcId, env)
	if err != nil {
		return VPCDetail{}, err
	}

	// Get VSwitches
	vswitches, err := ListVSwitches(region, vpcId, env)
	if err != nil {
		return VPCDetail{}, err
	}

	// Get Route Tables
	routeTables, err := ListRouteTables(region, vpcId, env)
	if err != nil {
		return VPCDetail{}, err
	}

	// Get Route Entries for each route table
	routeEntries := []RouteEntry{}
	for _, rt := range routeTables {
		entries, err := ListRouteEntries(region, rt.ID, env)
		if err != nil {
			continue
		}
		routeEntries = append(routeEntries, entries...)
	}

	return VPCDetail{
		VPC:          vpc,
		VSwitches:    vswitches,
		RouteTables:  routeTables,
		RouteEntries: routeEntries,
	}, nil
}

// Format functions
func FormatVPCs(items []VPC, output string) string {
	if output == "json" {
		b, _ := json.MarshalIndent(items, "", "  ")
		return string(b)
	}
	if len(items) == 0 {
		return "未找到 VPC"
	}
	lines := []string{fmt.Sprintf("VPC 数量: %d", len(items)), ""}
	for _, it := range items {
		name := it.Name
		if name == "" {
			name = it.ID
		}
		defaultTag := ""
		if it.IsDefault {
			defaultTag = " (默认)"
		}
		lines = append(lines, fmt.Sprintf("- %s%s", name, defaultTag))
		lines = append(lines, fmt.Sprintf("  ID: %s | CIDR: %s | 状态: %s", it.ID, it.CidrBlock, it.Status))
		if it.Description != "" {
			lines = append(lines, fmt.Sprintf("  描述: %s", it.Description))
		}
	}
	return strings.Join(lines, "\n")
}

func FormatVSwitches(items []VSwitch, output string) string {
	if output == "json" {
		b, _ := json.MarshalIndent(items, "", "  ")
		return string(b)
	}
	if len(items) == 0 {
		return "未找到交换机"
	}
	lines := []string{fmt.Sprintf("交换机数量: %d", len(items)), ""}
	for _, it := range items {
		name := it.Name
		if name == "" {
			name = it.ID
		}
		lines = append(lines, fmt.Sprintf("- %s", name))
		lines = append(lines, fmt.Sprintf("  ID: %s | CIDR: %s | 可用区: %s | 状态: %s", it.ID, it.CidrBlock, it.ZoneId, it.Status))
	}
	return strings.Join(lines, "\n")
}

func FormatRouteTables(items []RouteTable, output string) string {
	if output == "json" {
		b, _ := json.MarshalIndent(items, "", "  ")
		return string(b)
	}
	if len(items) == 0 {
		return "未找到路由表"
	}
	lines := []string{fmt.Sprintf("路由表数量: %d", len(items)), ""}
	for _, it := range items {
		name := it.Name
		if name == "" {
			name = it.ID
		}
		lines = append(lines, fmt.Sprintf("- %s (%s)", name, it.Type))
		lines = append(lines, fmt.Sprintf("  ID: %s | VPC: %s | VRouter: %s", it.ID, it.VpcId, it.VRouterId))
	}
	return strings.Join(lines, "\n")
}

func FormatRouteEntries(items []RouteEntry, output string) string {
	if output == "json" {
		b, _ := json.MarshalIndent(items, "", "  ")
		return string(b)
	}
	if len(items) == 0 {
		return "未找到路由条目"
	}
	lines := []string{fmt.Sprintf("路由条目数量: %d", len(items)), ""}
	lines = append(lines, "目标网段 | 下一跳类型 | 下一跳 ID | 类型")
	lines = append(lines, strings.Repeat("-", 60))
	for _, it := range items {
		lines = append(lines, fmt.Sprintf("%s | %s | %s | %s", it.DestinationCidr, it.NextHopType, it.NextHopId, it.Type))
	}
	return strings.Join(lines, "\n")
}

func FormatVPCDetail(detail VPCDetail, output string) string {
	if output == "json" {
		b, _ := json.MarshalIndent(detail, "", "  ")
		return string(b)
	}

	name := detail.VPC.Name
	if name == "" {
		name = detail.VPC.ID
	}
	defaultTag := ""
	if detail.VPC.IsDefault {
		defaultTag = " (默认)"
	}

	lines := []string{fmt.Sprintf("=== VPC: %s%s ===", name, defaultTag), ""}
	lines = append(lines, fmt.Sprintf("ID: %s", detail.VPC.ID))
	lines = append(lines, fmt.Sprintf("CIDR: %s | 状态: %s", detail.VPC.CidrBlock, detail.VPC.Status))
	lines = append(lines, fmt.Sprintf("VRouter: %s", detail.VPC.VRouterId))
	if detail.VPC.Description != "" {
		lines = append(lines, fmt.Sprintf("描述: %s", detail.VPC.Description))
	}

	// VSwitches
	lines = append(lines, "", fmt.Sprintf("=== 交换机 (%d) ===", len(detail.VSwitches)))
	if len(detail.VSwitches) == 0 {
		lines = append(lines, "无交换机")
	} else {
		for _, vs := range detail.VSwitches {
			vsName := vs.Name
			if vsName == "" {
				vsName = vs.ID
			}
			lines = append(lines, fmt.Sprintf("- %s: %s (%s)", vsName, vs.CidrBlock, vs.ZoneId))
		}
	}

	// Route Tables
	lines = append(lines, "", fmt.Sprintf("=== 路由表 (%d) ===", len(detail.RouteTables)))
	if len(detail.RouteTables) == 0 {
		lines = append(lines, "无路由表")
	} else {
		for _, rt := range detail.RouteTables {
			rtName := rt.Name
			if rtName == "" {
				rtName = rt.ID
			}
			lines = append(lines, fmt.Sprintf("- %s (%s)", rtName, rt.Type))
		}
	}

	// Route Entries
	lines = append(lines, "", fmt.Sprintf("=== 路由条目 (%d) ===", len(detail.RouteEntries)))
	if len(detail.RouteEntries) == 0 {
		lines = append(lines, "无路由条目")
	} else {
		lines = append(lines, "目标网段 | 下一跳类型 | 下一跳 ID | 类型")
		lines = append(lines, strings.Repeat("-", 60))
		for _, re := range detail.RouteEntries {
			lines = append(lines, fmt.Sprintf("%s | %s | %s | %s", re.DestinationCidr, re.NextHopType, re.NextHopId, re.Type))
		}
	}

	return strings.Join(lines, "\n")
}