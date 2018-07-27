package shell

import (
	"github.com/yunionio/jsonutils"
	"github.com/yunionio/mcclient"
	"github.com/yunionio/mcclient/modules"
)

func init() {
	type VCenterListOptions struct {
		BaseListOptions
	}
	R(&VCenterListOptions{}, "vcenter-list", "List VMWare vcenters", func(s *mcclient.ClientSession, args *VCenterListOptions) error {
		params := FetchPagingParams(args.BaseListOptions)
		result, err := modules.VCenters.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.VCenters.GetColumns(s))
		return nil
	})

	type VCenterCreateOptions struct {
		NAME   string `help:"Name of vcenter"`
		HOST   string `help:"Hostname or IP of vcenter server"`
		Port   int64  `help:"Port number" default:"443"`
		USER   string `help:"User account name"`
		PASSWD string `help:"Password"`
		Desc   string `help:"Description" metavar:"DESCRIPTION"`
	}
	R(&VCenterCreateOptions{}, "vcenter-create", "Create a vcenter", func(s *mcclient.ClientSession, args *VCenterCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString(args.HOST), "hostname")
		params.Add(jsonutils.NewString(args.USER), "account")
		params.Add(jsonutils.NewString(args.PASSWD), "password")
		if args.Port > 0 {
			params.Add(jsonutils.NewInt(args.Port), "port")
		}
		vc, err := modules.VCenters.Create(s, params)
		if err != nil {
			return err
		}
		printObject(vc)
		return nil
	})

	type VCenterDetailOptions struct {
		ID string `help:"ID or name of vcenter"`
	}

	R(&VCenterDetailOptions{}, "vcenter-show", "Show details of a vcenter", func(s *mcclient.ClientSession, args *VCenterDetailOptions) error {
		vc, err := modules.VCenters.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(vc)
		return nil
	})

	R(&VCenterDetailOptions{}, "vcenter-delete", "Delete a vcenter", func(s *mcclient.ClientSession, args *VCenterDetailOptions) error {
		vc, err := modules.VCenters.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(vc)
		return nil
	})

	type VCenteSyncOptions struct {
		ID           string `help:"Sync vcenter ID or name"`
		SyncHost     string `help:"Also full sync the host information"`
		SyncAllHosts bool   `help:"Sync all hosts"`
		Force        bool   `help:"Force sync, disregard status"`
	}
	R(&VCenteSyncOptions{}, "vcenter-sync", "Sync a vcenter", func(s *mcclient.ClientSession, args *VCenteSyncOptions) error {
		params := jsonutils.NewDict()
		if args.SyncAllHosts {
			params.Add(jsonutils.JSONTrue, "sync_host")
		} else if len(args.SyncHost) > 0 {
			params.Add(jsonutils.NewString(args.SyncHost), "sync_host_ip")
		}
		if args.Force {
			params.Add(jsonutils.JSONTrue, "force")
		}
		vc, err := modules.VCenters.PerformAction(s, args.ID, "sync", params)
		if err != nil {
			return err
		}
		printObject(vc)
		return nil
	})

	type VCenterUpdateCredentialOptions struct {
		ID       string `help:"ID or name of vcenter"`
		User     string `help:"New Account"`
		Password string `help:"New password"`
	}
	R(&VCenterUpdateCredentialOptions{}, "vcenter-update-credential", "Update account and password information of a vcenter", func(s *mcclient.ClientSession, args *VCenterUpdateCredentialOptions) error {
		params := jsonutils.NewDict()
		if len(args.User) > 0 {
			params.Add(jsonutils.NewString(args.User), "account")
		}
		if len(args.Password) > 0 {
			params.Add(jsonutils.NewString(args.Password), "password")
		}
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		vc, err := modules.VCenters.PerformAction(s, args.ID, "update-credential", params)
		if err != nil {
			return err
		}
		printObject(vc)
		return nil
	})

}
