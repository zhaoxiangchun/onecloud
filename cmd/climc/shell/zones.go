package shell

import (
	"github.com/yunionio/jsonutils"
	"github.com/yunionio/mcclient"
	"github.com/yunionio/mcclient/modules"
)

func init() {
	type ZoneListOptions struct {
		BaseListOptions
	}
	R(&ZoneListOptions{}, "zone-list", "List zones", func(s *mcclient.ClientSession, suboptions *ZoneListOptions) error {
		params := FetchPagingParams(suboptions.BaseListOptions)
		result, err := modules.Zones.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Zones.GetColumns(s))
		return nil
	})

	type ZoneUpdateOptions struct {
		ID       string `help:"ID or Name of zone to update"`
		Name     string `help:"Name of zone"`
		NameCN   string `help:"Name in Chinese"`
		Desc     string `metavar:"<DESCRIPTION>" help:"Description"`
		Location string `help:"Location"`
	}
	R(&ZoneUpdateOptions{}, "zone-update", "Update zone", func(s *mcclient.ClientSession, args *ZoneUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.NameCN) > 0 {
			params.Add(jsonutils.NewString(args.NameCN), "name_cn")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if len(args.Location) > 0 {
			params.Add(jsonutils.NewString(args.Location), "location")
		}
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		result, err := modules.Zones.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ZoneShowOptions struct {
		ID string `help:"ID or Name of the zone to show"`
	}
	R(&ZoneShowOptions{}, "zone-show", "Show zone details", func(s *mcclient.ClientSession, args *ZoneShowOptions) error {
		result, err := modules.Zones.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&ZoneShowOptions{}, "zone-delete", "Delete zone", func(s *mcclient.ClientSession, args *ZoneShowOptions) error {
		result, err := modules.Zones.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ZoneCreateOptions struct {
		NAME     string `help:"Name of zone"`
		NameCN   string `help:"Name in Chinese"`
		Desc     string `metavar:"<DESCRIPTION>" help:"Description"`
		Location string `help:"Location"`
	}
	R(&ZoneCreateOptions{}, "zone-create", "Create a zone", func(s *mcclient.ClientSession, args *ZoneCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		if len(args.NameCN) > 0 {
			params.Add(jsonutils.NewString(args.NameCN), "name_cn")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if len(args.Location) > 0 {
			params.Add(jsonutils.NewString(args.Location), "location")
		}
		zone, err := modules.Zones.Create(s, params)
		if err != nil {
			return err
		}
		printObject(zone)
		return nil
	})

}
