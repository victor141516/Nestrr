# Especificación visual de Nestrr

Versión visual: ronda 3
Manifiesto asociado: `screen-manifest.json` versión 3
Inventario validado: **85 PNG, 85 IDs, 0 archivos sin mapear**

## Dirección visual

Nestrr usa una estética cinematográfica sobria y completamente nueva:

- Fondo grafito y azul noche, superficies translúcidas oscuras y bordes de contraste bajo.
- Coral Nestrr para marca, acción primaria, progreso personal y foco activo.
- Menta para salud, disponibilidad, reproducción estable y confirmaciones.
- Ámbar para disponibilidad parcial, espera y degradación recuperable.
- Coral de alerta para errores, acciones destructivas y recuperación.
- Tipografía sans moderna, títulos de peso fuerte y números legibles para tiempo/progreso.
- Arte audiovisual ficticio coherente: **Órbita Ciega**, **Puerto Umbral** y **La Última Aurora**.

## Shell y navegación

En escritorio, la cabecera mantiene Nestrr, `Inicio / Buscar / Descargado`, salud de Plarr/Jackett/qBittorrent y avatar. En móvil, la barra inferior contiene únicamente `Inicio / Buscar / Descargado`; `Estado` y avatar son accesos secundarios superiores. El avatar de S03 móvil conduce a Ajustes mediante hotspot.

Las pantallas de reproducción eliminan la shell de catálogo y conservan una salida `Cerrar` visible. Ninguna pantalla móvil convierte Plarr, Jackett o qBittorrent en destinos principales.

## Contrato de estado único

Cada PNG representa un solo estado activable. No se mezclan en una misma lámina estados incompatibles como:

- listo + cargando;
- resultados + vacío;
- enviando + aceptado;
- éxito + error;
- menú + diálogo;
- toast de éxito + estado inicial o degradado.

Los sufijos `ready`, `loading`, `empty`, `error`, `sending`, `accepted`, `menu` y `dialog` identifican ramas independientes. `variantOf` expresa sólo parentesco semántico; la navegación usa IDs explícitos.

## Reglas de servicios y acciones

| Condición | Reproducción / descarga | Torrent | Mensaje |
|---|---|---|---|
| Todo sano y archivo en disco | visibles | visible | sin aviso |
| Película remota | ausentes | visible | `No descargada` |
| Torrent aceptado, aún sin archivo | ausentes | conserva contexto | `Pendiente de organizar` |
| Plarr degradado | ocultas | visible | aviso específico de Plarr |
| Jackett degradado | visibles | oculto | aviso específico de Jackett |
| qBittorrent degradado | visibles | oculto | aviso específico de qBittorrent |

S06 remoto, pendiente y degradados no contienen el toast inicial `Metadatos actualizados`.

## Continuidad del fixture

- La película remota de S04 es siempre **Órbita Ciega**. Cadena: `S04 → S06-remote → S11 → S06-pending → S05 → S06`.
- Rutas físicas en inglés; títulos de interfaz localizados.
- Puerto Umbral conserva **5/8 disponibles** y **3/8 vistos**. S01E02 está en disco con **38 % · 18:32**, no visto.
- Temporada 2 muestra error de carga sin inventar `0/6`.
- Rama inicial: `E01 → E02`; desde E02, E03 no está disponible y vuelve a S07 para buscar torrent.
- Autoplay independiente: `E05 → S10 → E06 · La baliza`. E06 no tiene siguiente.
- S10 sólo muestra autoplay; no incluye toast de episodio visto.

## Accesibilidad

- Acciones táctiles primarias amplias y separadas.
- Color acompañado por icono y etiqueta.
- Foco, selección y progreso con contraste sobre grafito.
- Espera con controles deshabilitados.
- Errores recuperables con `Reintentar`.
- Diálogos destructivos aislados, con cancelar y confirmar.

## Correspondencia ID → imagen

| ID | Pantalla / estado | Archivo | Dimensiones reales |
|---|---|---|---|
| S00 | Carga inicial | `assets/screens/s00-loading.png` | 1672×941 |
| S01 | Crear administrador · listo | `assets/screens/s01-bootstrap.png` | 1672×941 |
| S01-mobile | Crear administrador · listo · móvil | `assets/screens/s01-bootstrap-mobile.png` | 836×1882 |
| S01-loading | Crear administrador · enviando | `assets/screens/s01-bootstrap-loading.png` | 1672×941 |
| S01-loading-mobile | Crear administrador · enviando · móvil | `assets/screens/s01-bootstrap-loading-mobile.png` | 853×1844 |
| S01-error | Crear administrador · conflicto | `assets/screens/s01-bootstrap-error.png` | 1672×941 |
| S01-error-mobile | Crear administrador · conflicto · móvil | `assets/screens/s01-bootstrap-error-mobile.png` | 853×1844 |
| S02 | Iniciar sesión · listo | `assets/screens/s02-login.png` | 1672×941 |
| S02-mobile | Iniciar sesión · listo · móvil | `assets/screens/s02-login-mobile.png` | 853×1844 |
| S02-loading | Iniciar sesión · enviando | `assets/screens/s02-login-loading.png` | 1672×941 |
| S02-loading-mobile | Iniciar sesión · enviando · móvil | `assets/screens/s02-login-loading-mobile.png` | 853×1844 |
| S02-error | Iniciar sesión · error | `assets/screens/s02-login-error.png` | 1672×941 |
| S02-error-mobile | Iniciar sesión · error · móvil | `assets/screens/s02-login-error-mobile.png` | 853×1844 |
| S03 | Inicio | `assets/screens/s03-home.png` | 1672×941 |
| S03-mobile | Inicio · móvil | `assets/screens/s03-home-mobile.png` | 853×1844 |
| S04 | Buscar · resultados | `assets/screens/s04-search.png` | 1671×941 |
| S04-mobile | Buscar · resultados · móvil | `assets/screens/s04-search-mobile.png` | 853×1844 |
| S04-ready | Buscar · listo | `assets/screens/s04-search-ready.png` | 1672×941 |
| S04-ready-mobile | Buscar · listo · móvil | `assets/screens/s04-search-ready-mobile.png` | 853×1844 |
| S04-loading | Buscar · cargando | `assets/screens/s04-search-loading.png` | 1672×941 |
| S04-loading-mobile | Buscar · cargando · móvil | `assets/screens/s04-search-loading-mobile.png` | 852×1846 |
| S04-empty | Buscar · sin resultados | `assets/screens/s04-search-empty.png` | 1672×941 |
| S04-empty-mobile | Buscar · sin resultados · móvil | `assets/screens/s04-search-empty-mobile.png` | 853×1844 |
| S04-error | Buscar · error | `assets/screens/s04-search-error.png` | 1672×941 |
| S04-error-mobile | Buscar · error · móvil | `assets/screens/s04-search-error-mobile.png` | 853×1844 |
| S05 | Descargado | `assets/screens/s05-library.png` | 1672×941 |
| S05-mobile | Descargado · móvil | `assets/screens/s05-library-mobile.png` | 853×1844 |
| S06 | Detalle película · disponible | `assets/screens/s06-movie.png` | 1672×941 |
| S06-mobile | Detalle película · disponible · móvil | `assets/screens/s06-movie-mobile.png` | 853×1844 |
| S06-remote | Detalle película · remota | `assets/screens/s06-movie-remote.png` | 1672×941 |
| S06-remote-mobile | Detalle película · remota · móvil | `assets/screens/s06-movie-remote-mobile.png` | 853×1844 |
| S06-pending | Detalle película · pendiente | `assets/screens/s06-movie-pending.png` | 1672×941 |
| S06-pending-mobile | Detalle película · pendiente · móvil | `assets/screens/s06-movie-pending-mobile.png` | 853×1844 |
| S06-plarr-down | Detalle película · Plarr degradado | `assets/screens/s06-movie-plarr-down.png` | 1672×941 |
| S06-jackett-down | Detalle película · Jackett degradado | `assets/screens/s06-movie-jackett-down.png` | 1672×941 |
| S06-jackett-down-mobile | Detalle película · Jackett degradado · móvil | `assets/screens/s06-movie-jackett-down-mobile.png` | 853×1844 |
| S06-qbittorrent-down | Detalle película · qBittorrent degradado | `assets/screens/s06-movie-torrents-down.png` | 1672×941 |
| S07 | Detalle serie | `assets/screens/s07-show.png` | 1671×941 |
| S07-mobile | Detalle serie · móvil | `assets/screens/s07-show-mobile.png` | 853×1844 |
| S08 | Colección · lista | `assets/screens/s08-collection-ready.png` | 1672×941 |
| S08-mobile | Colección · lista · móvil | `assets/screens/s08-collection-ready-mobile.png` | 853×1844 |
| S08-empty | Colección · vacía | `assets/screens/s08-collection-empty.png` | 1672×941 |
| S08-empty-mobile | Colección · vacía · móvil | `assets/screens/s08-collection-empty-mobile.png` | 853×1844 |
| S08-error | Colección · error | `assets/screens/s08-collection-error.png` | 1672×941 |
| S08-error-mobile | Colección · error · móvil | `assets/screens/s08-collection-error-mobile.png` | 853×1844 |
| S09 | Player película | `assets/screens/s09-player.png` | 1672×941 |
| S09-mobile | Player película · móvil | `assets/screens/s09-player-mobile.png` | 853×1844 |
| S09-episode | Player episodio E02 | `assets/screens/s09-player-episode.png` | 1672×941 |
| S09-episode-mobile | Player episodio E02 · móvil | `assets/screens/s09-player-episode-mobile.png` | 853×1844 |
| S09-episode-e01 | Player episodio E01 | `assets/screens/s09-player-episode-e01.png` | 1672×941 |
| S09-episode-e01-mobile | Player episodio E01 · móvil | `assets/screens/s09-player-episode-e01-mobile.png` | 853×1844 |
| S09-episode-e05 | Player episodio E05 | `assets/screens/s09-player-episode-e05.png` | 1672×941 |
| S09-episode-e05-mobile | Player episodio E05 · móvil | `assets/screens/s09-player-episode-e05-mobile.png` | 853×1844 |
| S09-episode-e06 | Player episodio E06 | `assets/screens/s09-player-episode-e06.png` | 1672×941 |
| S09-episode-e06-mobile | Player episodio E06 · móvil | `assets/screens/s09-player-episode-e06-mobile.png` | 853×1844 |
| S10 | Autoplay E05 → E06 | `assets/screens/s10-autoplay.png` | 1672×941 |
| S10-mobile | Autoplay E05 → E06 · móvil | `assets/screens/s10-autoplay-mobile.png` | 853×1844 |
| S11 | Torrent · resultados | `assets/screens/s11-torrent-results.png` | 1672×941 |
| S11-mobile | Torrent · resultados · móvil | `assets/screens/s11-torrent-results-mobile.png` | 853×1844 |
| S11-ready | Torrent · listo | `assets/screens/s11-torrent-ready.png` | 1672×941 |
| S11-ready-mobile | Torrent · listo · móvil | `assets/screens/s11-torrent-ready-mobile.png` | 853×1844 |
| S11-loading | Torrent · cargando | `assets/screens/s11-torrent-loading.png` | 1672×941 |
| S11-loading-mobile | Torrent · cargando · móvil | `assets/screens/s11-torrent-loading-mobile.png` | 853×1844 |
| S11-sending | Torrent · enviando | `assets/screens/s11-torrent-sending.png` | 1672×941 |
| S11-sending-mobile | Torrent · enviando · móvil | `assets/screens/s11-torrent-sending-mobile.png` | 853×1844 |
| S11-accepted | Torrent · aceptado | `assets/screens/s11-torrent-accepted.png` | 1672×941 |
| S11-accepted-mobile | Torrent · aceptado · móvil | `assets/screens/s11-torrent-accepted-mobile.png` | 853×1844 |
| S11-error | Torrent · error | `assets/screens/s11-torrent-error.png` | 1672×941 |
| S11-error-mobile | Torrent · error · móvil | `assets/screens/s11-torrent-error-mobile.png` | 853×1844 |
| S12 | Ajustes · listo | `assets/screens/s12-settings-ready.png` | 1672×941 |
| S12-mobile | Ajustes · listo · móvil | `assets/screens/s12-settings-ready-mobile.png` | 853×1844 |
| S12-saved | Ajustes · guardado | `assets/screens/s12-settings-saved.png` | 1672×941 |
| S12-saved-mobile | Ajustes · guardado · móvil | `assets/screens/s12-settings-saved-mobile.png` | 853×1844 |
| S12-delete-dialog | Ajustes · confirmar borrado | `assets/screens/s12-settings-delete-dialog.png` | 1672×941 |
| S12-delete-dialog-mobile | Ajustes · confirmar borrado · móvil | `assets/screens/s12-settings-delete-dialog-mobile.png` | 853×1844 |
| S12-menu | Ajustes · menú de cuenta | `assets/screens/s12-settings-menu.png` | 1672×941 |
| S12-menu-mobile | Ajustes · menú de cuenta · móvil | `assets/screens/s12-settings-menu-mobile.png` | 853×1844 |
| S13 | Administración · listo | `assets/screens/s13-admin-ready.png` | 1672×941 |
| S13-mobile | Administración · listo · móvil | `assets/screens/s13-admin-ready-mobile.png` | 853×1844 |
| S13-duplicate-error | Administración · usuario duplicado | `assets/screens/s13-admin-duplicate-error.png` | 1672×941 |
| S13-duplicate-error-mobile | Administración · usuario duplicado · móvil | `assets/screens/s13-admin-duplicate-error-mobile.png` | 853×1844 |
| S13-member-menu | Administración · menú de miembro | `assets/screens/s13-admin-member-menu.png` | 1672×941 |
| S13-member-menu-mobile | Administración · menú de miembro · móvil | `assets/screens/s13-admin-member-menu-mobile.png` | 853×1844 |
| S13-reset-dialog | Administración · confirmar restablecimiento | `assets/screens/s13-admin-reset-dialog.png` | 1672×941 |
| S13-reset-dialog-mobile | Administración · confirmar restablecimiento · móvil | `assets/screens/s13-admin-reset-dialog-mobile.png` | 853×1844 |

## Validación de entrega

- JSON parseable.
- 85 IDs únicos y 85 PNG presentes.
- Dimensiones del manifiesto iguales a las reales.
- Todos los destinos de `hotspotSuggestions` resuelven.
- Ningún composite legacy S08/S11/S12/S13 permanece.
- Familias críticas separadas por estado en desktop y móvil.
- Identidad, servicios y continuidad del fixture comprobados visualmente.

## Limitaciones residuales

Son especificaciones raster high-fidelity, no capturas de una implementación. El texto secundario muy pequeño puede presentar variaciones tipográficas y debe sustituirse por texto real en frontend. Algunas imágenes conservan dimensiones cercanas, pero no idénticas, al lienzo nominal (por ejemplo 1671×941 o 852×1846); el manifiesto registra siempre el tamaño real y el prototipo debe escalar proporcionalmente.
