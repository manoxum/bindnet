import { useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

interface LogsPanelProps {
  title: string;
  path: string;
}

// LogsPanel le o stream de texto puro que o backend repassa do worker
// (docker logs -f) e vai anexando ao painel, com auto-scroll.
export function LogsPanel({ title, path }: LogsPanelProps) {
  const [lines, setLines] = useState("");
  const [following, setFollowing] = useState(false);
  const preRef = useRef<HTMLPreElement>(null);

  useEffect(() => {
    if (!following) return;
    const controller = new AbortController();
    setLines("");

    fetch(`/api${path}?tail=200&follow=true`, { credentials: "include", signal: controller.signal })
      .then(async (response) => {
        const reader = response.body?.getReader();
        if (!reader) return;
        const decoder = new TextDecoder();
        for (;;) {
          const { done, value } = await reader.read();
          if (done) break;
          setLines((current) => current + decoder.decode(value, { stream: true }));
        }
      })
      .catch(() => {
        // requisição abortada ao pausar/desmontar - comportamento esperado
      });

    return () => controller.abort();
  }, [following, path]);

  useEffect(() => {
    preRef.current?.scrollTo({ top: preRef.current.scrollHeight });
  }, [lines]);

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0">
        <CardTitle className="text-base">{title}</CardTitle>
        <Button size="sm" variant={following ? "destructive" : "outline"} onClick={() => setFollowing((v) => !v)}>
          {following ? "Parar" : "Acompanhar logs"}
        </Button>
      </CardHeader>
      <CardContent>
        <pre ref={preRef} className="h-64 overflow-auto rounded-md bg-black p-3 text-xs text-green-400">
          {lines || "Clique em 'Acompanhar logs' para ver a saída em tempo real."}
        </pre>
      </CardContent>
    </Card>
  );
}
