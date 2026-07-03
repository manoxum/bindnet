import { useEffect } from "react";
import { useOutletContext } from "react-router-dom";

export interface PageHeaderData {
  title: string;
  description?: string;
}

export type SetPageHeader = (header: PageHeaderData) => void;

// Cada página chama este hook para publicar seu título/descrição no header
// compartilhado do AppLayout (que os recebe via contexto do <Outlet/>).
export function usePageHeader(header: PageHeaderData) {
  const setHeader = useOutletContext<SetPageHeader>();

  useEffect(() => {
    setHeader(header);
  }, [setHeader, header.title, header.description]);
}
