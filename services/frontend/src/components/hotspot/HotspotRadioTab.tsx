import type { UseFormRegister } from "react-hook-form";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { SelectNative } from "@/components/ui/select-native";
import { TabsContent } from "@/components/ui/tabs";
import type { ConfigForm } from "@/components/hotspot/hotspot-schema";
import { WIFI_COUNTRIES } from "@/components/hotspot/wifi-countries";

interface HotspotRadioTabProps {
  register: UseFormRegister<ConfigForm>;
}

export function HotspotRadioTab({ register }: HotspotRadioTabProps) {
  return (
    <TabsContent value="radio" className="mt-0">
      <fieldset className="space-y-4">
        <legend className="text-sm font-medium text-muted-foreground">Rádio</legend>
        <div className="grid gap-4 sm:grid-cols-3">
          <div className="space-y-2">
            <Label htmlFor="WIFI_COUNTRY">País (código Wi-Fi)</Label>
            <SelectNative id="WIFI_COUNTRY" {...register("WIFI_COUNTRY")}>
              {WIFI_COUNTRIES.map((country) => (
                <option key={country.code} value={country.code}>
                  {country.name}
                </option>
              ))}
            </SelectNative>
          </div>
          <div className="space-y-2">
            <Label htmlFor="WIFI_FREQ_BAND">Banda</Label>
            <SelectNative id="WIFI_FREQ_BAND" {...register("WIFI_FREQ_BAND")}>
              <option value="auto">Automática</option>
              <option value="2.4">2.4GHz</option>
              <option value="5">5GHz</option>
            </SelectNative>
          </div>
          <div className="space-y-2">
            <Label htmlFor="WIFI_CHANNEL">Canal</Label>
            <Input id="WIFI_CHANNEL" placeholder="auto" {...register("WIFI_CHANNEL")} />
          </div>
          <div className="space-y-2 sm:col-span-3">
            <Label htmlFor="WIFI_CHANNEL_CANDIDATES">Canais candidatos</Label>
            <Input id="WIFI_CHANNEL_CANDIDATES" placeholder="1,6,11" {...register("WIFI_CHANNEL_CANDIDATES")} />
          </div>
        </div>
      </fieldset>
    </TabsContent>
  );
}
