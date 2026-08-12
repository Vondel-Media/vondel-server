import { cn } from "@/lib/utils";
import { useBranding } from "@/hooks/useBranding";

const VONDEL_WORDMARK_SRC = "/vondel-wordmark-sidebar.png";
const VONDEL_MARK_SRC = "/vondel-icon-1024.png";

export type VondelBrandVariant = "wordmark" | "mark";

interface VondelBrandProps {
  className?: string;
  imageClassName?: string;
  variant?: VondelBrandVariant;
}

export function VondelBrand({ className, imageClassName, variant = "wordmark" }: VondelBrandProps) {
  const isMark = variant === "mark";
  const { serverName, wordmarkUrl, markUrl } = useBranding();

  const customSrc = isMark ? markUrl : wordmarkUrl;
  const src = customSrc ?? (isMark ? VONDEL_MARK_SRC : VONDEL_WORDMARK_SRC);

  return (
    <span className={cn("block shrink-0", !isMark && "overflow-hidden", className)}>
      <img
        src={src}
        alt={serverName}
        className={cn("h-full w-full object-contain", isMark && "rounded-lg", imageClassName)}
      />
    </span>
  );
}
