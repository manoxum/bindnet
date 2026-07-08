import jsPDF from "jspdf";
import QRCode from "qrcode";
import { formatQuotaValue } from "@/components/hotspot/hotspot-limits-types";
import type { HotspotVoucher, HotspotVoucherBatch } from "@/components/hotspot/hotspot-voucher-types";

const COLUMNS = 2;
const ROWS_PER_PAGE = 7;
const MARGIN_X = 15;
const MARGIN_Y = 22;
const CARD_WIDTH = 90;
const CARD_HEIGHT = 30;
const GAP = 5;
const QR_SIZE = 24;

// Gera um PDF com um cartao por voucher do lote (codigo + QR code), em
// grade pronta para recortar - baixado pelo botao "Baixar PDF" na
// pagina de detalhe do lote (ver HotspotVoucherBatchDetail.tsx). Sem
// depender do dialogo de impressao do navegador. O QR encoda o mesmo
// texto do codigo, lido pelo leitor de camera no portal (ver
// src/components/portal/PortalVoucherQrScanner.tsx).
export async function downloadVoucherBatchPdf(batch: HotspotVoucherBatch, vouchers: HotspotVoucher[]) {
  const qrDataUrls = await Promise.all(vouchers.map((voucher) => QRCode.toDataURL(voucher.code, { margin: 0 })));

  const doc = new jsPDF({ unit: "mm", format: "a4" });
  const amountLabel = formatQuotaValue(batch.amountBytes, batch.amountUnit);
  const perPage = COLUMNS * ROWS_PER_PAGE;

  vouchers.forEach((voucher, index) => {
    const indexInPage = index % perPage;
    if (index > 0 && indexInPage === 0) {
      doc.addPage();
    }
    if (indexInPage === 0) {
      doc.setFontSize(14);
      doc.text(`Lote de vouchers ${batch.id} - ${amountLabel} cada`, MARGIN_X, 12);
    }

    const col = indexInPage % COLUMNS;
    const row = Math.floor(indexInPage / COLUMNS);
    const x = MARGIN_X + col * (CARD_WIDTH + GAP);
    const y = MARGIN_Y + row * (CARD_HEIGHT + GAP);
    const textX = x + QR_SIZE + 8;

    doc.setDrawColor(180);
    doc.rect(x, y, CARD_WIDTH, CARD_HEIGHT);
    doc.addImage(qrDataUrls[index], "PNG", x + 3, y + 3, QR_SIZE, QR_SIZE);

    doc.setFontSize(9);
    doc.text("Voucher hotspot", textX, y + 9);
    doc.setFontSize(12);
    doc.text(voucher.code, textX, y + 18);
    doc.setFontSize(9);
    doc.text(`Valor: ${amountLabel}`, textX, y + 25);
  });

  doc.save(`vouchers-${batch.id}.pdf`);
}
