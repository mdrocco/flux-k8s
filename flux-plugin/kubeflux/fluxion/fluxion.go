package fluxion

import (
	
	"github.com/cmisale/flux-sched/resource/hlapi/bindings/go/src/fluxcli"
	"github.com/flux-framework/flux-k8s/flux-plugin/kubeflux/utils" 
	"github.com/flux-framework/flux-k8s/flux-plugin/kubeflux/jobspec"
	pb "github.com/flux-framework/flux-k8s/flux-plugin/kubeflux/fluxcli-grpc"

	"context"
	"errors"
	"fmt"
	"io/ioutil"
)

type Fluxion struct {
	fctx 		*fluxcli.ReapiCtx
	pb.UnimplementedFluxcliServiceServer
}

func (f *Fluxion) Context() *fluxcli.ReapiCtx {
	return f.fctx
}

func (f *Fluxion) InitFluxion(policy *string, label *string) {
	f.fctx = fluxcli.NewReapiCli()

	fmt.Println("Created cli context ", f.fctx)
	fmt.Printf("%+v\n", f.fctx)
	filename := "/home/data/jgf/kubecluster.json"
	err := utils.CreateJGF(filename, label)
	if err != nil {
		return
	}
	
	jgf, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println("Error reading JGF")
		return
	}
	
	p := "{}"
	if *policy != "" {
		p = string("{\"matcher_policy\": \"" + *policy + "\"}")
		fmt.Println("Match policy: ", p)
	} 
	
	fluxcli.ReapiCliInit(f.fctx, string(jgf), p)

}

func (s *Fluxion) Cancel(ctx context.Context, in *pb.CancelRequest) (*pb.CancelResponse, error) {
	fmt.Printf("[GRPCServer] Received Cancel request %v\n", in)
	err := fluxcli.ReapiCliCancel(s.fctx, int64(in.JobID), true)
	if err < 0 {
		return nil, errors.New("Error in Cancel")
	}

	dr := &pb.CancelResponse{JobID: in.JobID, Error: int32(err)}
	fmt.Printf("[GRPCServer] Sending Cancel response %v\n", dr)

	fmt.Printf("[CancelRPC] Errors so far: %s\n", fluxcli.ReapiCliGetErrMsg(s.fctx))

	reserved, at, overhead, mode, fluxerr := fluxcli.ReapiCliInfo(s.fctx, int64(in.JobID))

	fmt.Println("\n\t----Job Info output---")
	fmt.Printf("jobid: %d\nreserved: %t\nat: %d\noverhead: %f\nmode: %s\nerror: %d\n", in.JobID, reserved, at, overhead, mode, fluxerr)

	fmt.Printf("[GRPCServer] Sending Cancel response %v\n", dr)
	return dr, nil
}

func (s *Fluxion) Match(ctx context.Context, in *pb.MatchRequest) (*pb.MatchResponse, error) {
	filename := "/home/data/jobspecs/jobspec.yaml"
	jobspec.CreateJobSpecYaml(in.Ps, in.Count, filename)

	spec, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.New("Error reading jobspec")
	}

	fmt.Printf("[GRPCServer] Received Match request %v\n", in)
	reserved, allocated, at, overhead, jobid, fluxerr := fluxcli.ReapiCliMatchAllocate(s.fctx, false, string(spec))
	utils.PrintOutput(reserved, allocated, at, overhead, jobid, fluxerr)

	fmt.Printf("[MatchRPC] Errors so far: %s\n", fluxcli.ReapiCliGetErrMsg(s.fctx))
	if fluxerr != 0 {
		return nil, errors.New("Error in ReapiCliMatchAllocate")
	}

	if allocated == "" {
		return nil, nil
	}

	nodetasks := utils.ParseAllocResult(allocated)
	
	nodetaskslist := make([]*pb.NodeAlloc, len(nodetasks))
	for i, result := range nodetasks {
		nodetaskslist[i] = &pb.NodeAlloc {
			NodeID: result.Basename,
			Tasks: int32(result.CoreCount)/in.Ps.Cpu,
		}
	}
	mr := &pb.MatchResponse{PodID: in.Ps.Id, Nodelist: nodetaskslist, JobID: int64(jobid)}
	fmt.Printf("[GRPCServer] Response %v \n", mr)
	return mr, nil
}
