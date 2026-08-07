// Muted, looping, autoplaying background video — cropped to ~15s and
// compressed for web (source clips were 20-40MB; these are ~1MB) so they
// load fast and never demand audio permission. A poster image shows
// instantly before the video's first frame decodes, and the whole thing
// degrades to just the poster if the browser can't/won't autoplay.
export function VideoBackground({ src, poster, overlayClassName }: { src: string; poster: string; overlayClassName?: string }) {
  return (
    <div className="absolute inset-0 overflow-hidden">
      <video
        autoPlay
        muted
        loop
        playsInline
        poster={poster}
        className="absolute inset-0 w-full h-full object-cover"
      >
        <source src={src} type="video/mp4" />
      </video>
      <div className={overlayClassName ?? "absolute inset-0 bg-gradient-to-r from-background via-background/70 to-background/30"} />
    </div>
  );
}
