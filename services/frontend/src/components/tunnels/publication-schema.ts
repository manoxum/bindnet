import { z } from "zod";
export const publicationSchema = z.object({
  name: z.string().trim().min(1, "Informe o nome do site ou serviço").max(100),
  domainId: z.string().min(1, "Escolha um domínio público nas opções acima"),
  subdomain: z.string().trim().max(63),
  localDomain: z
    .string()
    .trim()
    .min(1, "Informe o domínio DNS local do nginx")
    .max(253),
  method: z.enum(["https", "http"]),
  siteId: z.coerce.number().int().nonnegative(),
});
export type PublicationValues = z.infer<typeof publicationSchema>;
