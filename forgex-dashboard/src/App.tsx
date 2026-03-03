import { useEffect, useState, useRef } from "react";
import { Terminal, Copy, Loader2, Server, Coins, Zap } from "lucide-react";
import classNames from "classnames";

interface ProtocolMessage {
  id: string;
  sender: string;
  receiver: string;
  type: string;
  payload: any;
  timestamp: string;
}

interface CostUpdate {
  tokens: number;
  cost: number;
}

export default function App() {
  const [messages, setMessages] = useState<ProtocolMessage[]>([]);
  const [cost, setCost] = useState<CostUpdate>({ tokens: 0, cost: 0.0 });
  const [wsStatus, setWsStatus] = useState<"connecting" | "connected" | "disconnected">("connecting");
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let ws: WebSocket;
    let reconnectTimer: any;

    const connect = () => {
      setWsStatus("connecting");
      ws = new WebSocket("ws://localhost:8080/ws");

      ws.onopen = () => {
        setWsStatus("connected");
      };

      ws.onmessage = (event) => {
        try {
          const raw = JSON.parse(event.data);
          if (raw.topic === "sys_cost") {
            setCost({ tokens: raw.tokens, cost: raw.cost });
          } else if (raw.topic === "agent_event") {
            setMessages((prev) => {
              const next = [...prev, raw.data];
              if (next.length > 500) next.shift(); // keep last 500
              return next;
            });
          }
        } catch (err) {
          console.error("Failed to parse msg", err);
        }
      };

      ws.onclose = () => {
        setWsStatus("disconnected");
        reconnectTimer = setTimeout(connect, 3000);
      };
      
      ws.onerror = () => {
        ws.close();
      };
    };

    connect();

    return () => {
      clearTimeout(reconnectTimer);
      if (ws) ws.close();
    };
  }, []);

  // auto scroll
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [messages]);

  const renderPayload = (msg: ProtocolMessage) => {
    const { type, payload } = msg;
    if (type === "task") return `Assigned: ${payload.description}`;
    if (type === "result") {
      if (payload.success) return `✅ Task Success: ${payload.summary}`;
      else return `❌ Task Failed: ${payload.error}`;
    }
    if (type === "test") {
      const p = payload.passed ? "✅ Passed" : "❌ Failed";
      return `${p} - ${payload.failed_tests} failed out of ${payload.total_tests}`;
    }
    if (type === "status") {
      // Just string if it's broad status
      if (typeof payload === 'string') return payload;
      // Extract main status property
      if (payload.message) return payload.message;
    }
    return JSON.stringify(payload);
  };

  return (
    <div className="min-h-screen bg-neutral-900 text-neutral-300 font-sans selection:bg-cyan-900/40">
      
      {/* Header */}
      <header className="border-b border-neutral-800 bg-neutral-950/50 backdrop-blur-md sticky top-0 z-10 px-6 py-4 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="bg-gradient-to-tr from-cyan-600 to-emerald-500 rounded p-1.5 shadow-lg shadow-cyan-900/20">
            <Zap className="w-5 h-5 text-neutral-50" strokeWidth={2.5} />
          </div>
          <div>
            <h1 className="text-xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-neutral-50 to-neutral-400">ForgeX Dashboard</h1>
            <div className="flex items-center gap-2 mt-0.5">
              <span className={classNames(
                "w-2 h-2 rounded-full animate-pulse",
                wsStatus === "connected" ? "bg-emerald-500" : wsStatus === "connecting" ? "bg-amber-500" : "bg-red-500"
              )} />
              <span className="text-xs text-neutral-500 font-medium tracking-wide uppercase">
                {wsStatus === "connected" ? "Live Stream" : "Reconnecting..."}
              </span>
            </div>
          </div>
        </div>

        {/* Global Cost Monitors */}
        <div className="flex gap-4">
          <div className="bg-neutral-900/80 border border-neutral-800 rounded-lg px-4 py-2 flex items-center gap-3">
            <Coins className="w-4 h-4 text-emerald-500" />
            <div>
              <div className="text-[10px] text-neutral-500 uppercase tracking-widest font-bold">Total Cost</div>
              <div className="text-sm font-semibold text-neutral-200">${cost.cost.toFixed(4)}</div>
            </div>
          </div>
          
          <div className="bg-neutral-900/80 border border-neutral-800 rounded-lg px-4 py-2 flex items-center gap-3">
            <Server className="w-4 h-4 text-amber-500" />
            <div>
              <div className="text-[10px] text-neutral-500 uppercase tracking-widest font-bold">Tokens Processed</div>
              <div className="text-sm font-semibold text-neutral-200">{cost.tokens.toLocaleString()} tkns</div>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-6xl mx-auto p-6 grid grid-cols-1 lg:grid-cols-3 gap-6">

        {/* Left Col: Event Stream Feed */}
        <div className="lg:col-span-2 flex flex-col gap-4">
          <h2 className="text-sm font-bold text-neutral-400 uppercase tracking-wider flex items-center gap-2">
            <Terminal className="w-4 h-4" /> Agent Event Stream
          </h2>
          
          <div 
            ref={scrollRef}
            className="flex-1 bg-neutral-950 border border-neutral-800 rounded-xl max-h-[70vh] overflow-y-auto p-4 flex flex-col gap-3 scroll-smooth shadow-2xl relative"
          >
            {messages.length === 0 && (
              <div className="absolute inset-0 flex flex-col items-center justify-center text-neutral-600 gap-3">
                <Loader2 className="w-8 h-8 animate-spin text-neutral-800" />
                <p className="text-sm font-medium">Waiting for ForgeX core events...</p>
              </div>
            )}
            
            {messages.map((msg, i) => (
              <div 
                key={i} 
                className="group relative bg-[#131313] border border-[#222] rounded-lg p-3.5 pr-12 transition-all hover:border-neutral-700"
              >
                <div className="flex items-baseline gap-3 mb-1.5">
                  <span className="text-[10px] font-mono text-neutral-600 w-16 whitespace-nowrap">
                    {new Date(msg.timestamp).toLocaleTimeString([], { hour12: false, hour: '2-digit', minute:'2-digit', second:'2-digit' })}
                  </span>
                  <span className={classNames(
                    "text-xs font-bold uppercase tracking-wider px-2 py-0.5 rounded-sm",
                    msg.sender === "supervisor" ? "bg-fuchsia-950/40 text-fuchsia-400 border border-fuchsia-900/50" :
                    msg.sender === "coder" ? "bg-cyan-950/40 text-cyan-400 border border-cyan-900/50" :
                    msg.sender === "tester" ? "bg-rose-950/40 text-rose-400 border border-rose-900/50" :
                    "bg-neutral-800/50 text-neutral-400"
                  )}>
                    {msg.sender || "System"}
                  </span>
                  <span className="text-neutral-600 text-xs">{"-->"}</span>
                  <span className="text-xs text-neutral-500 font-mono capitalize">{msg.receiver || "Broadcast"}</span>
                  <span className="ml-auto text-[10px] px-1.5 rounded-sm font-bold text-neutral-500 bg-neutral-900 border border-neutral-800">
                    {msg.type}
                  </span>
                </div>
                
                <div className="pl-14 text-sm font-mono text-neutral-300 whitespace-pre-wrap break-words">
                  {renderPayload(msg)}
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Right Col: Active Agents */}
        <div className="flex flex-col gap-4">
           <h2 className="text-sm font-bold text-neutral-400 uppercase tracking-wider">
            Agent Cluster
          </h2>
          <div className="bg-neutral-950 border border-neutral-800 rounded-xl p-5 flex flex-col gap-4">
            
            {/* Sup */}
            <div className="flex items-center gap-4 p-3 rounded-lg border border-fuchsia-900/30 bg-fuchsia-950/10 relative overflow-hidden">
               <div className="absolute left-0 top-0 bottom-0 w-1 bg-fuchsia-500 shadow-[0_0_12px_theme(colors.fuchsia.500)]"></div>
               <div className="w-10 h-10 rounded-full bg-fuchsia-900/50 flex items-center justify-center border border-fuchsia-500/30">
                 <span className="text-xs font-bold text-fuchsia-400">SUP</span>
               </div>
               <div>
                  <div className="text-sm font-bold text-neutral-200">Supervisor</div>
                  <div className="text-xs text-neutral-500 font-medium">Orchestrating tasks</div>
               </div>
               <div className="ml-auto flex items-center">
                  <span className="w-2 h-2 rounded-full bg-fuchsia-400 animate-pulse shadow-[0_0_8px_theme(colors.fuchsia.400)]"></span>
               </div>
            </div>

            {/* Coder */}
            <div className="flex items-center gap-4 p-3 rounded-lg border border-cyan-900/30 bg-cyan-950/10 relative overflow-hidden">
               <div className="absolute left-0 top-0 bottom-0 w-1 bg-cyan-500 shadow-[0_0_12px_theme(colors.cyan.500)]"></div>
               <div className="w-10 h-10 rounded-full bg-cyan-900/50 flex items-center justify-center border border-cyan-500/30">
                 <span className="text-xs font-bold text-cyan-400">DEV</span>
               </div>
               <div>
                  <div className="text-sm font-bold text-neutral-200">Coder</div>
                  <div className="text-xs text-neutral-500 font-medium">Listening for sub-tasks</div>
               </div>
               <div className="ml-auto flex items-center">
                  <span className="w-2 h-2 rounded-full bg-cyan-400 animate-pulse shadow-[0_0_8px_theme(colors.cyan.400)]"></span>
               </div>
            </div>

            {/* Tester */}
            <div className="flex items-center gap-4 p-3 rounded-lg border border-rose-900/30 bg-rose-950/10 relative overflow-hidden">
               <div className="absolute left-0 top-0 bottom-0 w-1 bg-rose-500 shadow-[0_0_12px_theme(colors.rose.500)]"></div>
               <div className="w-10 h-10 rounded-full bg-rose-900/50 flex items-center justify-center border border-rose-500/30">
                 <span className="text-xs font-bold text-rose-400">TST</span>
               </div>
               <div>
                  <div className="text-sm font-bold text-neutral-200">Tester</div>
                  <div className="text-xs text-neutral-500 font-medium">Awaiting builds</div>
               </div>
               <div className="ml-auto flex items-center">
                  <span className="w-2 h-2 rounded-full bg-rose-400 animate-pulse shadow-[0_0_8px_theme(colors.rose.400)]"></span>
               </div>
            </div>

          </div>
        </div>

      </main>
    </div>
  );
}
