self.importScripts("wasm_exec.js");

if (!WebAssembly.instantiateStreaming) {
  // polyfill
  WebAssembly.instantiateStreaming = async (resp, importObject) => {
    const source = await (await resp).arrayBuffer();
    return await WebAssembly.instantiate(source, importObject);
  };
}

// @ts-ignore
const go = new Go();
WebAssembly.instantiateStreaming(fetch('/main.wasm'), go.importObject)
  .then((result) => {
    go.run(result.instance);
    console.log("aggregator loaded okay");
    postMessage({ type: AggResponse.Ready });
  }).catch((e) => {
    console.error(e);
    postMessage({
      type: AggResponse.Failed,
      reason: e instanceof Error ? e.message : "Unknown Error" });
  });

// @ts-ignore
function initialize(req: { cfg: string }) {
  const resp = initializeAggregator(req.cfg);
  if (resp != null) {
    return { type: AggResponse.Failed, reason: JSON.parse(resp).error };
  }
  // TODO: return deserialized typed parsed config?
  return { type: AggResponse.Initialized };
}

function add(req: { result: Uint8Array, itr: number }) {
  const resp = aggregate(req.result, req.itr);
  if (resp != null) {
    return { type: AggResponse.Failed, reason: JSON.parse(resp).error };
  }
  return { type: AggResponse.Done };
}

function doFlush(req: {}) {
  // TODO: have a specific result response type to enforce (protos?)
  const resp = JSON.parse(flush());
  if (resp.error) {
    return { type: AggResponse.Failed, reason: resp.error };
  }
  return { type: AggResponse.Result, result: resp };
}

function validate(req: { cfg: string }) {
  const resp = JSON.parse(validateConfig(req.cfg));
  if (resp.error) {
    return { type: AggResponse.Failed, reason: resp.error };
  }
  return { type: AggResponse.Validate, cfg: resp };
}

// @ts-ignore
function handleRequest(req: any): any {
  switch (req.type as AggRequest) {
    case AggRequest.Initialize:
      return initialize(req);
    case AggRequest.Add:
      return add(req);
    case AggRequest.Flush:
      return doFlush(req);
    case AggRequest.Validate:
      return validate(req);
    case AggRequest.BuildInfo:
      // TODO:
      return;
    default:
      console.error("aggregator - unknown request: ", req);
      throw new Error("aggregator unknown request");
  }
}
onmessage = (ev) => postMessage(handleRequest(ev.data));

// TODO: I hate this
// Web Workers do not currently support modules (in all browsers), so instead all the relevant code in common
// has to be copy/pasted over
// Clean up when supported: https://developer.mozilla.org/en-US/docs/Web/JavaScript/Guide/Modules

enum AggRequest {
  Initialize = "initialize",
  Add = "add",
  Flush = "flush",
  Validate = "validate",
  BuildInfo = "build_info",
}

enum AggResponse {
  Failed = "failed",
  Ready = "ready",
  Initialized = "initialized",
  Done = "done",
  Result = "result",
  Validate = "validated",
  BuildInfo = "build_info",
}