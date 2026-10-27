"use client";

import Image from "next/image";
import { useState } from "react";
import Lightbox from "yet-another-react-lightbox";
import Zoom from "yet-another-react-lightbox/plugins/zoom";
import "yet-another-react-lightbox/styles.css";

import { Main } from "@/components/layout/main";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

import { DashboardShell } from "@/modules/dashboard";
import { useTeacherAuth } from "@/modules/identity";

const ARCHITECTURE_IMAGE_SRC =
  "/assets/diagrams/architecture/architecture.png";
const LIFECYCLE_IMAGE_SRC = "/assets/diagrams/lifecycle/content-lifecycle.svg";

const diagrams = [
  {
    key: "architecture",
    title: "Architecture",
    image: ARCHITECTURE_IMAGE_SRC,
    imageWidth: 3840,
    imageHeight: 2160,
    alt: "ClassGame current modular architecture diagram",
  },
  {
    key: "lifecycle",
    title: "Lifecycle",
    image: LIFECYCLE_IMAGE_SRC,
    imageWidth: 1920,
    imageHeight: 1080,
    alt: "ClassGame content lifecycle diagram",
  },
] as const;

type Diagram = (typeof diagrams)[number];

export function ArchitecturePage() {
  const auth = useTeacherAuth();
  const [active, setActive] = useState<Diagram["key"]>("architecture");
  const [lightboxOpen, setLightboxOpen] = useState(false);

  const current =
    diagrams.find((diagram) => diagram.key === active) ?? diagrams[0];

  async function handleSignOut() {
    await auth.signOutTeacher();
  }

  return (
    <DashboardShell user={auth.user!} onSignOut={handleSignOut}>
      <Main id="main-content">
        <Tabs
          value={active}
          onValueChange={(value) => {
            setActive(value as Diagram["key"]);
          }}
          className="gap-4"
        >
          <TabsList className="grid w-full grid-cols-2 sm:w-fit">
            {diagrams.map((diagram) => (
              <TabsTrigger key={diagram.key} value={diagram.key}>
                {diagram.title}
              </TabsTrigger>
            ))}
          </TabsList>

          <TabsContent value="architecture">
            <DiagramImage
              diagram={diagrams[0]}
              onOpen={() => setLightboxOpen(true)}
            />
          </TabsContent>

          <TabsContent value="lifecycle">
            <DiagramImage
              diagram={diagrams[1]}
              onOpen={() => setLightboxOpen(true)}
            />
          </TabsContent>
        </Tabs>

        <ArchitectureLightbox
          open={lightboxOpen}
          onClose={() => setLightboxOpen(false)}
          image={{
            src: current.image,
            alt: current.alt,
            width: current.imageWidth,
            height: current.imageHeight,
          }}
        />
      </Main>
    </DashboardShell>
  );
}

function DiagramImage({
  diagram,
  onOpen,
}: {
  readonly diagram: Diagram;
  readonly onOpen: () => void;
}) {
  return (
    <div className="relative w-full overflow-hidden rounded-xl bg-white p-2 md:p-4">
      <button
        type="button"
        aria-label={diagram.alt}
        onClick={onOpen}
        className="block w-full cursor-zoom-in rounded-lg outline-none focus-visible:ring-2 focus-visible:ring-purple-400 focus-visible:ring-offset-2 focus-visible:ring-offset-white"
      >
        <Image
          key={diagram.key}
          src={diagram.image}
          alt={diagram.alt}
          width={diagram.imageWidth}
          height={diagram.imageHeight}
          className="h-auto w-full object-contain"
          loading="lazy"
          draggable={false}
        />
      </button>
    </div>
  );
}

function ArchitectureLightbox({
  image,
  onClose,
  open,
}: {
  readonly open: boolean;
  readonly image: {
    readonly src: string;
    readonly alt: string;
    readonly width: number;
    readonly height: number;
  };
  readonly onClose: () => void;
}) {
  return (
    <Lightbox
      open={open}
      close={onClose}
      slides={[image]}
      plugins={[Zoom]}
      carousel={{ finite: true, padding: 0, spacing: 0 }}
      controller={{ disableSwipeNavigation: true }}
      render={{
        buttonPrev: () => null,
        buttonNext: () => null,
      }}
      animation={{ zoom: 300 }}
      zoom={{
        maxZoomPixelRatio: 3,
        scrollToZoom: true,
        doubleClickMaxStops: 3,
        keyboardMoveDistance: 80,
      }}
    />
  );
}
