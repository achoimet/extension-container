// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package extcontainer

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-container/pkg/container/runc"
	"github.com/steadybit/extension-container/pkg/networkutils"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
)

func NewNetworkCorruptContainerAction(r runc.Runc) action_kit_sdk.Action[NetworkActionState] {
	return &networkAction{
		optsProvider: corruptPackages(r),
		optsDecoder:  corruptPackagesDecode,
		description:  getNetworkCorruptDescription(),
		runc:         r,
	}
}

func getNetworkCorruptDescription() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          fmt.Sprintf("%s.network_package_corruption", targetID),
		Label:       "Package Corruption",
		Description: "Inject corrupt packets by introducing single bit error at a random offset into network traffic.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(targetIcon),
		TargetSelection: &action_kit_api.TargetSelection{
			TargetType:         targetID,
			SelectionTemplates: &targetSelectionTemplates,
		},
		Category:    extutil.Ptr("network"),
		Kind:        action_kit_api.Attack,
		TimeControl: action_kit_api.External,
		Parameters: append(
			commonNetworkParameters,
			action_kit_api.ActionParameter{
				Name:         "networkCorruption",
				Label:        "Package Corruption",
				Description:  extutil.Ptr("How much of the traffic should be corrupted?"),
				Type:         action_kit_api.Percentage,
				DefaultValue: extutil.Ptr("15"),
				Required:     extutil.Ptr(true),
				Order:        extutil.Ptr(1),
			},
			action_kit_api.ActionParameter{
				Name:        "networkInterface",
				Label:       "Network Interface",
				Description: extutil.Ptr("Target Network Interface which should be attacked."),
				Type:        action_kit_api.StringArray,
				Required:    extutil.Ptr(false),
				Order:       extutil.Ptr(104),
			},
		),
	}
}

func corruptPackages(r runc.Runc) networkOptsProvider {
	return func(ctx context.Context, request action_kit_api.PrepareActionRequestBody) (networkutils.Opts, error) {
		containerId := request.Target.Attributes["container.id"][0]
		corruption := extutil.ToUInt(request.Config["networkCorrupt"])

		var restrictedUrls []string
		if request.ExecutionContext != nil && request.ExecutionContext.RestrictedUrls != nil {
			restrictedUrls = *request.ExecutionContext.RestrictedUrls
		}

		filter, err := mapToNetworkFilter(ctx, r, containerId, request.Config, restrictedUrls)
		if err != nil {
			return nil, err
		}

		interfaces := extutil.ToStringArray(request.Config["networkInterface"])
		if len(interfaces) == 0 {
			interfaces, err = readNetworkInterfaces(ctx, r, RemovePrefix(containerId))
			if err != nil {
				return nil, err
			}
		}

		if len(interfaces) == 0 {
			return nil, fmt.Errorf("no network interfaces specified")
		}

		return &networkutils.CorruptPackagesOpts{
			Filter:     filter,
			Corruption: corruption,
			Interfaces: interfaces,
		}, nil
	}
}

func corruptPackagesDecode(data json.RawMessage) (networkutils.Opts, error) {
	var opts networkutils.CorruptPackagesOpts
	err := json.Unmarshal(data, &opts)
	return &opts, err
}