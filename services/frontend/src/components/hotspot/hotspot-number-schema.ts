import { z } from "zod";

// Fonte unica dos campos numericos dos formularios do hotspot. Eles sao
// <input> de texto (ver HotspotRateFields.tsx), entao chegam como string
// crua e cada schema repetia a mesma validacao - isto substitui a copia
// que existia em hotspot-device-limits-schema.ts, hotspot-profile-schema.ts
// e device-detail/DeviceCreditCard.tsx.

// A UI e em portugues e o operador digita no teclado do proprio
// celular, entao "1,5" e tao natural quanto "1.5" - normalizeDecimal
// aceita as duas formas e devolve a que o Number() entende (Number("1,5")
// e NaN, ou seja, sem isto uma virgula viraria "numero invalido").
export function normalizeDecimal(value: string): string {
  return value.trim().replace(",", ".");
}

export function parseDecimal(value: string): number {
  return Number(normalizeDecimal(value));
}

const DECIMAL_PATTERN = /^\d+(\.\d+)?$/;

// Valor positivo com fracao opcional; string vazia = campo em branco
// (sem limite), nunca um erro. Usado por taxa e cota (1.5GB, 17.5KB/s):
// a fracao sobrevive ate o fim nos dois casos - taxa e double precision
// no Postgres desde a migration 20260716000000_hotspot_rate_decimal (e o
// tc parseia decimal), e cota sempre foi gravada em bytes inteiros
// (quotaValueToBytes arredonda o produto), entao so a validacao do
// formulario a recusava.
export const optionalPositiveDecimal = z
  .string()
  .trim()
  .refine(
    (value) => value === "" || (DECIMAL_PATTERN.test(normalizeDecimal(value)) && parseDecimal(value) > 0),
    "Deve ser um número positivo (aceita decimal, ex.: 1.5)",
  );

// Valor inteiro positivo, opcional - continua valendo onde a fracao
// ainda nao foi pedida (politica de credito: recarga/plafond em GB).
export const optionalPositiveInt = z
  .string()
  .trim()
  .refine((value) => value === "" || (/^\d+$/.test(value) && Number(value) > 0), "Deve ser um número positivo");
