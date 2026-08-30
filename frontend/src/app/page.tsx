import { TopNav } from "@/components/shell/top-nav";
import { Footer } from "@/components/shell/footer";
import { PageContainer } from "@/components/shell/page-container";
import { HeroSection } from "@/components/landing/hero-section";
import { ProblemsSolutionsSection } from "@/components/landing/problems-solutions-section";
import { UspSection } from "@/components/landing/usp-section";
import { JourneySection } from "@/components/landing/journey-section";

/** `/` — the Transverse landing page, pixel-traced from Figma `61:46`. */
export default function Home() {
  return (
    <div className="flex min-h-full flex-col bg-tv-bg">
      <TopNav />
      <main className="flex flex-1 flex-col">
        <PageContainer>
          <HeroSection />
        </PageContainer>
        <PageContainer>
          <ProblemsSolutionsSection />
        </PageContainer>
        <PageContainer>
          <UspSection />
        </PageContainer>
        <PageContainer>
          <JourneySection />
        </PageContainer>
      </main>
      <Footer />
    </div>
  );
}
