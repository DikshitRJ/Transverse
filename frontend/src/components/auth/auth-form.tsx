"use client";

import { useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Loader2, Mail, Lock, User, Eye, EyeOff, AlertCircle } from "lucide-react";
import { useAuth } from "@/components/providers/auth-provider";
import { postSignInDestination } from "@/components/auth/user-status";
import { getMe } from "@/lib/api/endpoints";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { OAuthButtons } from "@/components/auth/oauth-buttons";
import { cn } from "@/lib/utils";

type AuthMode = "signin" | "signup";

export function AuthForm({ className }: { className?: string }) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const nextParam = searchParams.get("next");

  const { login, register } = useAuth();
  const [mode, setMode] = useState<AuthMode>("signin");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [username, setUsername] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    const cleanEmail = email.trim();
    const cleanPassword = password.trim();
    const cleanUsername = username.trim();

    if (!cleanEmail || !cleanPassword) {
      setError("Please fill in both email and password.");
      return;
    }

    if (cleanPassword.length < 6) {
      setError("Password must be at least 6 characters.");
      return;
    }

    setLoading(true);

    try {
      if (mode === "signin") {
        await login(cleanEmail, cleanPassword);
      } else {
        await register(cleanEmail, cleanPassword, cleanUsername || undefined);
      }

      // Determine where to navigate
      if (nextParam && nextParam.startsWith("/")) {
        router.push(nextParam);
        router.refresh();
      } else {
        const user = await getMe().catch(() => null);
        const dest = user ? postSignInDestination(user) : (mode === "signup" ? "/onboarding" : "/dashboard");
        router.push(dest);
        router.refresh();
      }
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Authentication failed. Please try again.";
      setError(message);
      setLoading(false);
    }
  };

  return (
    <div className={cn("flex flex-col gap-6", className)}>
      {/* Mode Switcher Tabs */}
      <div className="grid grid-cols-2 rounded-lg border border-tv-border bg-tv-surface-deep/80 p-1">
        <button
          type="button"
          onClick={() => {
            setMode("signin");
            setError(null);
          }}
          className={cn(
            "flex items-center justify-center rounded-md py-2 font-mono text-xs font-semibold uppercase tracking-wider transition-all",
            mode === "signin"
              ? "bg-tv-cyan/20 text-tv-cyan shadow-sm glow-text-cyan"
              : "text-tv-text-body hover:text-tv-text-hi",
          )}
        >
          Sign In
        </button>
        <button
          type="button"
          onClick={() => {
            setMode("signup");
            setError(null);
          }}
          className={cn(
            "flex items-center justify-center rounded-md py-2 font-mono text-xs font-semibold uppercase tracking-wider transition-all",
            mode === "signup"
              ? "bg-tv-cyan/20 text-tv-cyan shadow-sm glow-text-cyan"
              : "text-tv-text-body hover:text-tv-text-hi",
          )}
        >
          Create Account
        </button>
      </div>

      {/* Error Alert */}
      {error && (
        <div
          role="alert"
          className="flex items-start gap-2.5 rounded-lg border border-tv-rose/40 bg-tv-rose/10 p-3 text-xs text-tv-rose"
        >
          <AlertCircle className="size-4 shrink-0 mt-0.5" />
          <span className="leading-relaxed">{error}</span>
        </div>
      )}

      {/* Form */}
      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        {mode === "signup" && (
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="auth-username" className="font-mono text-xs text-tv-text-hi">
              Username <span className="text-tv-text-body font-normal">(optional)</span>
            </Label>
            <div className="relative">
              <User className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-tv-text-body pointer-events-none" />
              <Input
                id="auth-username"
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="e.g. codemaster"
                disabled={loading}
                className="pl-9 h-10 border-tv-border bg-tv-surface-deep text-sm text-tv-text-hi placeholder:text-tv-text-body/60 focus-visible:border-tv-cyan focus-visible:ring-tv-cyan/30"
              />
            </div>
          </div>
        )}

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="auth-email" className="font-mono text-xs text-tv-text-hi">
            Email ID <span className="text-tv-cyan">*</span>
          </Label>
          <div className="relative">
            <Mail className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-tv-text-body pointer-events-none" />
            <Input
              id="auth-email"
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="you@example.com"
              disabled={loading}
              className="pl-9 h-10 border-tv-border bg-tv-surface-deep text-sm text-tv-text-hi placeholder:text-tv-text-body/60 focus-visible:border-tv-cyan focus-visible:ring-tv-cyan/30"
            />
          </div>
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="auth-password" className="font-mono text-xs text-tv-text-hi">
            Password <span className="text-tv-cyan">*</span>
          </Label>
          <div className="relative">
            <Lock className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-tv-text-body pointer-events-none" />
            <Input
              id="auth-password"
              type={showPassword ? "text" : "password"}
              required
              minLength={6}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              disabled={loading}
              className="pl-9 pr-10 h-10 border-tv-border bg-tv-surface-deep text-sm text-tv-text-hi placeholder:text-tv-text-body/60 focus-visible:border-tv-cyan focus-visible:ring-tv-cyan/30"
            />
            <button
              type="button"
              onClick={() => setShowPassword(!showPassword)}
              aria-label={showPassword ? "Hide password" : "Show password"}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-tv-text-body hover:text-tv-text-hi transition-colors"
            >
              {showPassword ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
            </button>
          </div>
        </div>

        <Button
          type="submit"
          disabled={loading}
          size="lg"
          className="mt-2 w-full h-10 justify-center gap-2 normal-case font-semibold"
        >
          {loading && <Loader2 className="size-4 animate-spin" />}
          {mode === "signin" ? "Sign In" : "Create Account"}
        </Button>
      </form>

      {/* Divider */}
      <div className="relative flex items-center justify-center">
        <div className="absolute inset-0 flex items-center">
          <div className="w-full border-t border-tv-border" />
        </div>
        <span className="relative bg-tv-surface px-3 font-mono text-[11px] text-tv-text-body uppercase tracking-wider">
          Or continue with
        </span>
      </div>

      {/* GitHub OAuth Button */}
      <OAuthButtons />
    </div>
  );
}
