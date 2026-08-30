import Image from "next/image";
import { Button } from "@/components/ui/button";
import { oauthRedirectPath } from "@/lib/api/endpoints";

/**
 * GitHub OAuth button navigating to `GET /api/v1/auth/oauth/github/redirect`.
 */
export function OAuthButtons({ className }: { className?: string }) {
  return (
    <div className={className}>
      <Button
        render={<a href={oauthRedirectPath("github")} />}
        className="w-full justify-start gap-3 normal-case"
        size="lg"
      >
        <span className="flex size-6 items-center justify-center overflow-hidden rounded-full bg-white">
          <Image
            src="/figma/github-mark.png"
            alt=""
            width={24}
            height={24}
            className="size-full object-cover"
          />
        </span>
        Continue with GitHub
      </Button>
    </div>
  );
}
