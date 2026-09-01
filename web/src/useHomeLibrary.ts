import { ref, type Ref } from "vue";
import { api, session } from "./api";

export interface ContinueItem {
  progress: any;
  title: string;
  episodeTitle?: string;
  imagePath?: string;
  to: string;
}

interface HomeLibraryOptions {
  progress: Ref<any[]>;
  title: (media: any) => string;
}

export function useHomeLibrary(options: HomeLibraryOptions) {
  const library = ref<any[]>([]);
  const continueItems = ref<ContinueItem[]>([]);

  async function loadContinueItems(allProgress = options.progress.value) {
    const candidates = allProgress
      .filter((item) => item.started && !item.watched)
      .slice(0, 8);
    const shows = new Map<number, Promise<any>>();
    const showDetails = (id: number) => {
      if (!shows.has(id)) {
        shows.set(
          id,
          api(`/catalog/tv/${id}?language=${session.user?.language}`),
        );
      }
      return shows.get(id)!;
    };
    const resolved = await Promise.all(
      candidates.map(async (item): Promise<ContinueItem | null> => {
        try {
          if (item.mediaType === "movie") {
            const movie = await api<any>(
              `/catalog/movie/${item.tmdbId}?language=${session.user?.language}`,
            );
            return {
              progress: item,
              title: options.title(movie),
              imagePath: movie.backdrop_path || movie.poster_path,
              to: `/movie/${item.tmdbId}`,
            };
          }
          if (item.mediaType === "episode" && item.showTmdbId) {
            const [show, episode] = await Promise.all([
              showDetails(item.showTmdbId),
              api<any>(
                `/catalog/tv/${item.showTmdbId}/season/${item.seasonNumber}/episode/${item.episodeNumber}?language=${session.user?.language}`,
              ),
            ]);
            return {
              progress: item,
              title: options.title(show),
              episodeTitle: options.title(episode),
              imagePath:
                episode.still_path || show.backdrop_path || show.poster_path,
              to: `/tv/${item.showTmdbId}?season=${item.seasonNumber}&episode=${item.episodeNumber}`,
            };
          }
        } catch {
          return null;
        }
        return null;
      }),
    );
    continueItems.value = resolved.filter(Boolean) as ContinueItem[];
  }

  async function loadLibrary() {
    const response = await api<any>("/library");
    library.value = response.items || [];
  }

  return { continueItems, library, loadContinueItems, loadLibrary };
}
