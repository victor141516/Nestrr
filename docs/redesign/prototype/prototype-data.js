/* Coordinates are percentages of each generated screen. */
window.NESTRR_PROTOTYPE = {
  scenarios: {
    S00: {
      label: "Simular condición inicial",
      options: [
        { label: "Instancia nueva", target: "S01" },
        { label: "Sin sesión", target: "S02" },
        { label: "Sesión válida", target: "S03" }
      ]
    },
    S06: {
      label: "Simular salud de servicios",
      options: [
        { label: "Todo sano", target: "S06" },
        { label: "Plarr caído", target: "S06-plarr-down" },
        { label: "Jackett caído", target: "S06-jackett-down" },
        { label: "qBittorrent caído", target: "S06-qbittorrent-down" }
      ]
    }
  },
  flows: {
    auth: {
      title: "Acceso y descubrimiento",
      description: "Primera visita, autenticación y llegada al catálogo.",
      steps: ["S00", "S01", "S01-loading", "S03", "S04-ready", "S04-loading", "S04"]
    },
    movie: {
      title: "Película: de remota a reproducible",
      description: "Buscar, enviar a qBittorrent, comprobar la biblioteca y reproducir.",
      steps: ["S03", "S04", "S06-remote", "S11-ready", "S11-loading", "S11", "S11-sending", "S11-accepted", "S06-pending", "S05", "S06", "S09"]
    },
    show: {
      title: "Serie parcial y siguiente episodio",
      description: "Reanuda S01E02, detecta que S01E03 falta y abre su búsqueda torrent.",
      steps: ["S03", "S07", "S09-episode", "S07", "S11-ready"]
    },
    autoplay: {
      title: "Autoplay de episodio disponible",
      description: "Finaliza S01E05, confirma S01E06 y conserva navegación episódica.",
      steps: ["S09-episode-e05", "S10", "S09-episode-e06", "S07"]
    },
    admin: {
      title: "Administración y suplantación",
      description: "Gestiona miembros, actúa como uno de ellos y vuelve al administrador.",
      steps: ["S03", "S12", "S12-menu", "S13", "S13-member-menu", "S13-reset-dialog", "S13", "S13-member-menu", "S03", "S12", "S12-menu", "S03"]
    },
    slow: {
      title: "Reproducción con conexión lenta",
      description: "Reanuda, cae a Auto 240p, muestra un fallo recuperable y conserva el progreso.",
      steps: ["S06", "S09", "S09", "S06"]
    },
    degraded: {
      title: "Servicios degradados",
      description: "Comprueba que cada integración desactiva únicamente las acciones que realmente dependen de ella.",
      steps: ["S06", "S06-plarr-down", "S06", "S06-jackett-down", "S06", "S06-qbittorrent-down", "S06"]
    }
  },
  semantic: {
    S00: { summary: "Nestrr decide si debe mostrar bootstrap, login o la aplicación autenticada.", facts: { Estado: "Comprobando sesión y servicios; el fallo de red ofrece Reintentar.", Destinos: "Instancia nueva → Crear administrador; sin sesión → Login; refresh válido → Inicio." } },
    S01: { summary: "Primera ejecución: la primera cuenta válida se convierte en la única administradora.", facts: { Campos: "Usuario (2–40), contraseña (mínimo 10), idioma inicial.", Estados: "Listo, validación, conflicto de bootstrap y envío." } },
    S02: { summary: "Acceso privado a una instancia configurada, sin registro público.", facts: { Campos: "Usuario y contraseña.", Error: "Las credenciales inválidas no revelan qué campo falló." } },
    S03: { summary: "Descubrimiento con destacado, progreso personal y catálogo mixto.", facts: { Disponibilidad: "Disponible, parcial, no descargado y comprobando.", Progreso: "Por usuario; no equivale a disponibilidad en disco.", Servicios: "Plarr, Jackett y qBittorrent se expresan con texto, icono y color." } },
    S04: { summary: "Búsqueda TMDB por películas o series con disponibilidad progresiva.", facts: { Filtros: "Tipo y Sólo descargado.", Estados: "Inicial, buscando, resultados, vacío y error.", Idioma: "Metadatos en el idioma del usuario; rutas físicas no se localizan." } },
    S05: { summary: "Biblioteca organizada verificada directamente en el sistema de archivos.", facts: { Contenido: "Películas, episodios y filas sin identidad TMDB.", Autoridad: "Estar aquí significa presente en disco, no sólo descargándose.", Rutas: "Nombres canónicos ingleses aunque el título visible esté localizado." } },
    S06: { summary: "Detalle de película con disponibilidad compartida y progreso personal separados.", facts: { Archivo: "Disponible en disco.", Progreso: "42 % visto; acción Reanudar.", Descargas: "Original y 1080p/720p/480p/360p/240p en streaming.", Torrent: "Sólo si Jackett y qBittorrent están sanos." } },
    S07: { summary: "Serie parcialmente descargada; cada episodio conserva su propio estado y acción.", facts: { Disponibilidad: "5/8 episodios en disco.", Progreso: "3/8 vistos (38 %), dato personal separado.", Error: "Temporada 2 puede reintentarse sin perder Temporada 1." } },
    S08: { summary: "Colección de películas sin reproducción ni torrent agregados.", facts: { Acciones: "Abrir cada parte o refrescar metadatos.", Estados: "Con partes, vacía y error de carga como estados alternativos." } },
    S09: { summary: "Reproductor Plarr en conexión lenta con fallo recuperable.", facts: { Calidad: "Empieza en Auto; aquí cae a 240p.", Progreso: "53:12 de 2:11:08; se conserva ante error.", Pistas: "Audio español y subtítulos desactivados." } },
    "S09-episode": { summary: "Reproductor de episodio con contexto, navegación y progreso propios.", facts: { Episodio: "Puerto Umbral · S01E02 · Marea muerta.", Progreso: "18:32 de 48:10; 38 % visto.", Navegación: "Anterior y siguiente conservan el contexto de la serie." } },
    S10: { summary: "Autoplay episódico tras marcar como visto el episodio actual.", facts: { Cuenta: "5 segundos con Reproducir ahora y Cancelar.", Navegación: "Anterior/siguiente atraviesa temporadas cuando procede." } },
    S11: { summary: "Jackett busca y qBittorrent recibe; aceptar no equivale a estar disponible.", facts: { Consulta: "Editable y en inglés/original; categoría independiente.", Filtros: "Semillas, trackers, incluir y excluir; locales.", Resultado: "Enviando, enviado o reintentable tras fallo; no hay porcentaje posterior." } },
    S12: { summary: "Preferencias personales de idioma y limpieza acotada de metadata.", facts: { Idioma: "Cambia UI/metadatos y audio inicial, nunca nombres físicos.", Borrado: "Sólo documentos TMDB propios; no archivos, imágenes ni progreso.", Suplantación: "La banda global permite volver al administrador." } },
    S13: { summary: "Administración de miembros y lectura segura del estado del servidor.", facts: { Usuarios: "Crear, renombrar, resetear, suplantar y eliminar miembros.", Seguridad: "Resetear revoca sesiones; el administrador único no se elimina.", Servidor: "Estado y rutas visibles sin exponer secretos." } }
    ,"S06-remote": { summary: "Película conocida por TMDB pero todavía ausente de la biblioteca local.", facts: { Disponibilidad: "No descargada.", Acciones: "Torrent y metadatos; no ofrece reproducir ni descargar.", Integraciones: "Jackett y qBittorrent están sanos." } }
    ,"S06-pending": { summary: "qBittorrent aceptó el torrent, pero Nestrr aún no dispone del archivo.", facts: { Estado: "Pendiente de aparecer en la biblioteca organizada.", Verdad: "Aceptado no implica descargado ni reproducible.", Acción: "Comprobar biblioteca sin inventar porcentaje." } }
    ,"S06-plarr-down": { summary: "La reproducción y las descargas están temporalmente desactivadas porque Plarr no responde.", facts: { Disponible: "El archivo sigue en disco.", Permitido: "Torrent y metadatos.", Recuperación: "Reintentar sólo el servicio afectado." } }
    ,"S06-torrents-down": { summary: "La película sigue reproducible aunque Jackett o qBittorrent no estén disponibles.", facts: { Disponible: "Reanudar y descargar permanecen activos.", Bloqueado: "Buscar torrent.", Recuperación: "Reintentar las integraciones de torrent." } }
    ,"S06-jackett-down": { summary: "Jackett no responde, pero el archivo local y Plarr siguen disponibles.", facts: { Disponible: "Reanudar y descargar permanecen activos.", Bloqueado: "Buscar torrent porque no se pueden consultar índices.", Recuperación: "Reintentar únicamente Jackett." } }
    ,"S06-jackett-down-mobile": { summary: "Jackett degradado en móvil sin afectar la reproducción local.", facts: { Disponible: "Reanudar y descargar permanecen activos.", Bloqueado: "Buscar torrent.", Recuperación: "Reintentar Jackett y conservar el viewport móvil." } }
    ,"S06-qbittorrent-down": { summary: "qBittorrent no responde, pero el archivo local y Plarr siguen disponibles.", facts: { Disponible: "Reanudar y descargar permanecen activos.", Bloqueado: "Enviar un torrent nuevo.", Recuperación: "Reintentar únicamente qBittorrent." } }
  },
  hotspots: {
    S00: [
      { action: "Abrir bootstrap", target: "S01", x: 43, y: 30, w: 4.8, h: 17 },
      { action: "Abrir login", target: "S02", x: 47.8, y: 30, w: 4.8, h: 17 },
      { action: "Continuar con sesión válida", target: "S03", x: 52.6, y: 30, w: 4.8, h: 17 },
      { action: "Reintentar conexión", target: "S00", x: 36.3, y: 77, w: 9.2, h: 6.4, message: "Conexión reintentada: comprobando sesión y servicios…" }
    ],
    S01: [
      { action: "Crear administrador", target: "S03", x: 54.5, y: 71.5, w: 36.5, h: 7 },
      { action: "Iniciar sesión tras conflicto", target: "S02", x: 78.7, y: 83.2, w: 11.5, h: 5.8 }
    ],
    S02: [
      { action: "Iniciar sesión", target: "S03", x: 59.2, y: 56, w: 30, h: 7.6 }
    ],
    S03: [
      { action: "Buscar", target: "S04", x: 18.2, y: 1.6, w: 8.2, h: 6 },
      { action: "Abrir Descargado", target: "S05", x: 26.4, y: 1.6, w: 10.6, h: 6 },
      { action: "Abrir ajustes", target: "S12", x: 94, y: 1.2, w: 4.5, h: 6.5 },
      { action: "Ver detalles de Órbita Ciega", target: "S06", x: 2.7, y: 38.8, w: 11.5, h: 6.7 },
      { action: "Reanudar Ecos del Horizonte", target: "S06", x: 2.8, y: 51, w: 46.2, h: 13 },
      { action: "Abrir Puerto Umbral", target: "S07", x: 2.5, y: 69.2, w: 15, h: 27 }
    ],
    S04: [
      { action: "Volver a Inicio", target: "S03", x: 11, y: 1.5, w: 6.7, h: 6 },
      { action: "Buscar aurora", target: "S04", x: 80.5, y: 22.7, w: 16.7, h: 7.5, message: "Búsqueda completada: 7 resultados; comprobando disponibilidad…" },
      { action: "Elegir Películas", target: "S04", x: 2.7, y: 32.1, w: 9.2, h: 5.4, message: "Tipo de búsqueda: Películas" },
      { action: "Elegir Series", target: "S04", x: 12.1, y: 32.1, w: 9.1, h: 5.4, message: "Tipo de búsqueda: Series" },
      { action: "Abrir La Última Aurora", target: "S06", x: 2.8, y: 50.1, w: 13.8, h: 33 },
      { action: "Abrir Puerto Umbral", target: "S07", x: 43.7, y: 50.1, w: 12.2, h: 33 },
      { action: "Abrir Bajo la Aurora no descargada", target: "S06-remote", x: 57, y: 50.1, w: 13.8, h: 33 }
    ],
    S05: [
      { action: "Volver a Inicio", target: "S03", x: 11, y: 1.4, w: 6.7, h: 6 },
      { action: "Enumerar de nuevo", target: "S05", x: 28.8, y: 25.2, w: 4.2, h: 6.1, message: "Biblioteca enumerada: los archivos nuevos ya pueden aparecer." },
      { action: "Abrir Órbita Ciega", target: "S06", x: 3, y: 38, w: 94, h: 10.6 },
      { action: "Abrir Puerto Umbral S01E02", target: "S07", x: 3, y: 48.7, w: 94, h: 10.2 },
      { action: "Reintentar raíz fallida", target: "S05", x: 23.8, y: 90, w: 8.4, h: 6.5, message: "Raíz reintentada. El resto de la biblioteca permanece visible." }
    ],
    S06: [
      { action: "Volver a Inicio", target: "S03", x: 9, y: 1.5, w: 6.8, h: 5.7 },
      { action: "Reanudar película", target: "S09", x: 21.7, y: 45, w: 10.5, h: 6.3 },
      { action: "Elegir descarga", target: "S06", x: 32.6, y: 45, w: 9.1, h: 6.3, message: "Menú de descarga abierto: Original y renditions MP4 en streaming." },
      { action: "Buscar torrent", target: "S11", x: 42, y: 45, w: 10.2, h: 6.3 },
      { action: "Actualizar metadatos", target: "S06", x: 52.4, y: 45, w: 12.3, h: 6.3, message: "Metadatos actualizados correctamente." },
      { action: "Abrir Saga Horizonte", target: "S08", x: 2, y: 52.5, w: 18, h: 11 },
      { action: "Abrir recomendación", target: "S06", x: 2, y: 67, w: 13.2, h: 24, message: "Detalle de la recomendación abierto (simulado con la misma plantilla)." }
    ],
    S07: [
      { action: "Volver a Inicio", target: "S03", x: 9.8, y: 1.5, w: 6.4, h: 5.7 },
      { action: "Buscar torrent de la serie", target: "S11", x: 17.7, y: 44.4, w: 11.5, h: 5.8 },
      { action: "Alternar Temporada 1", target: "S07", x: 3, y: 51, w: 94, h: 5.6, message: "Temporada 1 contraída/expandida sin perder sus estados." },
      { action: "Reproducir S01E01", target: "S09-episode", x: 81.5, y: 57, w: 11.8, h: 6 },
      { action: "Reanudar S01E02", target: "S09-episode", x: 81.5, y: 64.2, w: 11.8, h: 6 },
      { action: "Buscar torrent S01E03", target: "S11", x: 81.5, y: 71.7, w: 11.8, h: 6 },
      { action: "Reintentar Temporada 2", target: "S07", x: 43, y: 92.5, w: 18.5, h: 5.3, message: "Temporada reintentada de forma localizada; la Temporada 1 sigue disponible." }
    ],
    S08: [
      { action: "Volver a Inicio", target: "S03", x: 11, y: 1.4, w: 6.7, h: 6 },
      { action: "Actualizar metadatos", target: "S08", x: 18.6, y: 38.1, w: 13.9, h: 5.8, message: "Metadatos de la colección actualizados." },
      { action: "Abrir Ecos del Horizonte", target: "S06", x: 2.8, y: 50.8, w: 30.9, h: 21 },
      { action: "Abrir Fragmentos de Luz", target: "S06", x: 34.2, y: 50.8, w: 30.8, h: 21 },
      { action: "Abrir Órbita Ciega", target: "S06", x: 65.6, y: 50.8, w: 31, h: 21 },
      { action: "Reintentar títulos", target: "S08", x: 67.8, y: 90.6, w: 10.2, h: 5.1, message: "Se reintenta sólo la carga de títulos de la saga." }
    ],
    S09: [
      { action: "Cerrar player", target: "S06", x: 2.5, y: 2.6, w: 7.8, h: 5.2 },
      { action: "Reintentar reproducción", target: "S09", x: 39.3, y: 48, w: 21.4, h: 6.2, message: "Reconectando en Auto · 240p; el progreso se conserva." },
      { action: "Volver al detalle", target: "S06", x: 39.3, y: 55.2, w: 21.4, h: 6.2 },
      { action: "Cambiar calidad", target: "S09", x: 64.3, y: 49.8, w: 12.5, h: 29, message: "Calidad manual seleccionada. Volver a Auto sigue disponible." },
      { action: "Simular episodio terminado", target: "S10", x: 53, y: 84.2, w: 45, h: 12 }
    ],
    S10: [
      { action: "Cerrar player", target: "S07", x: 2, y: 3.2, w: 7.2, h: 5.6 },
      { action: "Reproducir ahora", target: "S09-episode", x: 65.5, y: 62.3, w: 27.6, h: 6.2 },
      { action: "Cancelar autoplay", target: "S09-episode", x: 65.5, y: 68.6, w: 27.6, h: 5.7 },
      { action: "Ir al episodio anterior", target: "S09-episode", x: 39.7, y: 90.1, w: 8.5, h: 5.8 }
    ],
    S11: [
      { action: "Cerrar modal", target: "S06", x: 77.5, y: 9.5, w: 3.2, h: 5 },
      { action: "Buscar torrents", target: "S11", x: 56.4, y: 25.2, w: 7.7, h: 6.6, message: "Jackett devolvió 4 resultados ordenados por semillas." },
      { action: "Cambiar filtros", target: "S11", x: 20.5, y: 33.5, w: 58.5, h: 14.5, message: "Filtros aplicados localmente; no se relanza la búsqueda." },
      { action: "Enviar resultado", target: "S06-pending", x: 69.8, y: 59.7, w: 8.5, h: 5, message: "Aceptado por qBittorrent. El contenido todavía no está disponible en Nestrr." },
      { action: "Reintentar envío", target: "S11", x: 69.8, y: 75.2, w: 8.5, h: 5, message: "Envío reintentado. Si qBittorrent lo acepta, se confirmará sin inventar progreso." },
      { action: "Comprobar biblioteca", target: "S05", x: 20.6, y: 18, w: 59, h: 5.5 }
    ],
    S12: [
      { action: "Volver al administrador", target: "S03", x: 84.5, y: 0.5, w: 13.5, h: 5.5, clearFlag: "impersonating" },
      { action: "Guardar idioma", target: "S12", x: 4.8, y: 76.1, w: 13.6, h: 6.4, message: "Preferencias guardadas; nombres físicos siguen en inglés." },
      { action: "Cancelar borrado", target: "S12", x: 70.2, y: 75.9, w: 7.2, h: 6.2, message: "Borrado cancelado; no se modificó ningún dato." },
      { action: "Confirmar borrado", target: "S12", x: 78.5, y: 75.9, w: 13.3, h: 6.2, message: "Sólo se borraron tus documentos TMDB; archivos, imágenes y progreso no cambian." },
      { action: "Abrir Administración", target: "S13", x: 84.5, y: 24.6, w: 11.5, h: 5.2 },
      { action: "Cerrar sesión", target: "S02", x: 84.5, y: 31, w: 11.5, h: 5.2 }
    ],
    S13: [
      { action: "Volver a Inicio", target: "S03", x: 11, y: 1.4, w: 6.7, h: 6 },
      { action: "Añadir miembro", target: "S13", x: 46.5, y: 36.2, w: 10.2, h: 5.1, message: "Miembro creado. El formulario se limpia y la lista se actualiza." },
      { action: "Abrir acciones de Bruno", target: "S13", x: 53.8, y: 55.8, w: 3, h: 5, message: "Menú de Bruno abierto: renombrar, resetear, suplantar o eliminar." },
      { action: "Restablecer contraseña", target: "S13", x: 33.3, y: 86.7, w: 8, h: 5.2, message: "Contraseña restablecida; se revocaron las sesiones de Bruno." },
      { action: "Ver como Bruno", target: "S03", x: 47, y: 70.2, w: 11, h: 5.2, message: "Suplantación iniciada. La banda persiste hasta volver al administrador.", setFlag: "impersonating" }
    ],
    "S02-mobile": [
      { action: "Iniciar sesión", target: "S03-mobile", x: 8, y: 57, w: 84, h: 7.2 }
    ],
    "S03-mobile": [
      { action: "Alternar panel de servicios", target: "S03-mobile", x: 65.5, y: 1.5, w: 18, h: 4.5, message: "Panel de servicios abierto/cerrado. Los estados siguen accesibles en móvil." },
      { action: "Ver detalles", target: "S06-mobile", x: 4, y: 28.3, w: 92, h: 5.5 },
      { action: "Abrir Puerto Umbral", target: "S07-mobile", x: 3.7, y: 47.8, w: 45, h: 17 },
      { action: "Buscar", target: "S04", x: 34, y: 95, w: 28, h: 4.5 },
      { action: "Descargado", target: "S05", x: 62, y: 95, w: 33, h: 4.5 }
    ],
    "S06-mobile": [
      { action: "Volver a Inicio", target: "S03-mobile", x: 3.5, y: 0.5, w: 14, h: 4.3 },
      { action: "Reanudar", target: "S09-mobile", x: 5.5, y: 42.4, w: 89, h: 5.1 },
      { action: "Alternar descarga", target: "S06-mobile", x: 5.5, y: 47, w: 38, h: 4.7, message: "Bottom sheet de descarga abierto/cerrado." },
      { action: "Buscar torrent", target: "S11-mobile", x: 46, y: 47, w: 37, h: 4.7 },
      { action: "Abrir Saga Horizonte", target: "S08", x: 5.5, y: 51.6, w: 89, h: 6.3 },
      { action: "Cerrar descarga", target: "S06-mobile", x: 86, y: 67.3, w: 8.5, h: 4.5, message: "Bottom sheet cerrado; el foco vuelve a Descargar." }
    ],
    "S07-mobile": [
      { action: "Buscar torrent de la serie", target: "S11-mobile", x: 5.3, y: 39.1, w: 40, h: 4.2 },
      { action: "Alternar temporada", target: "S07-mobile", x: 3.4, y: 44, w: 93, h: 4.8, message: "Temporada 1 contraída/expandida." },
      { action: "Reproducir S01E01", target: "S09-episode-mobile", x: 6, y: 57.4, w: 88, h: 3.7 },
      { action: "Reanudar S01E02", target: "S09-episode-mobile", x: 6, y: 70.2, w: 88, h: 3.7 },
      { action: "Buscar S01E03", target: "S11-mobile", x: 6, y: 83.6, w: 88, h: 3.7 },
      { action: "Volver a Inicio", target: "S03-mobile", x: 1, y: 94.6, w: 16, h: 5.3 }
    ],
    "S09-mobile": [
      { action: "Cerrar player", target: "S06-mobile", x: 2.5, y: 1.3, w: 18, h: 4.3 },
      { action: "Cambiar audio", target: "S09-mobile", x: 7, y: 61, w: 86, h: 5.2, message: "Pista predeterminada: Español; la preferencia explícita prevalece." },
      { action: "Cambiar calidad", target: "S09-mobile", x: 7, y: 66.4, w: 86, h: 5.2, message: "Auto selecciona 240p para esta conexión lenta." },
      { action: "Cambiar subtítulos", target: "S09-mobile", x: 7, y: 72, w: 86, h: 5.2, message: "Selector de subtítulos abierto." }
    ],
    "S11-mobile": [
      { action: "Cerrar torrent", target: "S06-mobile", x: 85.5, y: 2.5, w: 10, h: 4.4 },
      { action: "Buscar", target: "S11-mobile", x: 7, y: 26.7, w: 86, h: 5.4, message: "Búsqueda ejecutada sin ocultar los resultados por el teclado." },
      { action: "Alternar filtros", target: "S11-mobile", x: 7, y: 32.2, w: 86, h: 5.2, message: "Filtros expandidos/contraídos." },
      { action: "Enviar resultado", target: "S06-pending", x: 9, y: 52.2, w: 82, h: 4.2, message: "Aceptado por qBittorrent. Todavía no está disponible en Nestrr." }
    ],
    "S01-mobile": [
      { action: "Crear administrador", target: "S03-mobile", x: 10.5, y: 75.8, w: 79, h: 4.7 },
      { action: "Iniciar sesión tras conflicto", target: "S02-mobile", x: 64, y: 85.4, w: 26, h: 4.5 }
    ],
    "S04-mobile": [
      { action: "Buscar aurora", target: "S04-mobile", x: 4.5, y: 18.8, w: 91, h: 4.2, message: "Búsqueda completada: comprobando 3 de 8 títulos…" },
      { action: "Elegir Películas", target: "S04-mobile", x: 4.5, y: 24.3, w: 45, h: 3.8, message: "Tipo de búsqueda: Películas" },
      { action: "Elegir Series", target: "S04-mobile", x: 50, y: 24.3, w: 45.5, h: 3.8, message: "Tipo de búsqueda: Series" },
      { action: "Abrir La Última Aurora", target: "S06-mobile", x: 4.5, y: 36.5, w: 45, h: 22.5 },
      { action: "Abrir Puerto Umbral", target: "S07-mobile", x: 50.5, y: 36.5, w: 45, h: 22.5 },
      { action: "Ir a Inicio", target: "S03-mobile", x: 4.5, y: 93.5, w: 29, h: 5 },
      { action: "Ir a Descargado", target: "S05-mobile", x: 65.5, y: 93.5, w: 30, h: 5 }
    ],
    "S05-mobile": [
      { action: "Enumerar de nuevo", target: "S05-mobile", x: 45, y: 18.2, w: 10, h: 4.5, message: "Biblioteca enumerada: los archivos nuevos ya pueden aparecer." },
      { action: "Abrir Órbita Ciega", target: "S06-mobile", x: 4.5, y: 23, w: 91, h: 17.5 },
      { action: "Abrir Puerto Umbral S01E02", target: "S07-mobile", x: 4.5, y: 41, w: 91, h: 17.5 },
      { action: "Reintentar raíz fallida", target: "S05-mobile", x: 4.5, y: 88.2, w: 91, h: 4.5, message: "Raíz reintentada sin ocultar el resto de la biblioteca." },
      { action: "Ir a Inicio", target: "S03-mobile", x: 4, y: 94, w: 29, h: 5 },
      { action: "Ir a Buscar", target: "S04-mobile", x: 34, y: 94, w: 30, h: 5 }
    ],
    "S10-mobile": [
      { action: "Cerrar player", target: "S07-mobile", x: 3.5, y: 1.3, w: 19, h: 4.2 },
      { action: "Reproducir ahora", target: "S09-episode-mobile", x: 7, y: 77, w: 86, h: 4.5 },
      { action: "Cancelar autoplay", target: "S09-episode-mobile", x: 7, y: 82.1, w: 86, h: 4.5 },
      { action: "Ir al episodio anterior", target: "S09-episode-mobile", x: 3.5, y: 91, w: 25, h: 5 },
      { action: "Ir al episodio siguiente", target: "S09-episode-mobile", x: 71.5, y: 91, w: 25, h: 5 }
    ],
    "S12-mobile": [
      { action: "Volver al administrador", target: "S03-mobile", x: 62, y: 0, w: 38, h: 4.2, clearFlag: "impersonating" },
      { action: "Abrir Administración", target: "S13-mobile", x: 60, y: 13.5, w: 35, h: 4.5 },
      { action: "Cerrar sesión", target: "S02-mobile", x: 60, y: 18.5, w: 35, h: 4.5 },
      { action: "Guardar idioma", target: "S12-mobile", x: 8, y: 48.8, w: 84, h: 4.5, message: "Preferencias guardadas; los nombres físicos siguen en inglés." },
      { action: "Cancelar borrado", target: "S12-mobile", x: 11, y: 84, w: 38, h: 4.5, message: "Borrado cancelado." },
      { action: "Confirmar borrado", target: "S12-mobile", x: 51, y: 84, w: 37, h: 4.5, message: "Sólo se borraron tus documentos TMDB." }
    ],
    "S13-mobile": [
      { action: "Añadir miembro", target: "S13-mobile", x: 7.5, y: 34.5, w: 85, h: 4.2, message: "Miembro creado; la lista se actualiza." },
      { action: "Abrir acciones de Bruno", target: "S13-mobile", x: 82, y: 52.8, w: 10, h: 4.5, message: "Menú de Bruno abierto." },
      { action: "Restablecer contraseña", target: "S13-mobile", x: 49, y: 77.5, w: 35, h: 4.5, message: "Contraseña restablecida; se revocaron sus sesiones." },
      { action: "Ver como Bruno", target: "S03-mobile", x: 48, y: 61.5, w: 35, h: 4.5, message: "Suplantación iniciada.", setFlag: "impersonating" },
      { action: "Volver a Inicio", target: "S03-mobile", x: 3, y: 94, w: 30, h: 5 }
    ],
    "S09-episode": [
      { action: "Cerrar player", target: "S07", x: 2, y: 2, w: 8, h: 5.2 },
      { action: "Episodio anterior", target: "S09-episode", x: 38.5, y: 72.5, w: 10, h: 6, message: "Episodio anterior cargado conservando la temporada." },
      { action: "Episodio siguiente", target: "S09-episode", x: 52, y: 72.5, w: 10, h: 6, message: "Episodio siguiente cargado conservando la temporada." },
      { action: "Alternar pistas", target: "S09-episode", x: 57, y: 86, w: 40, h: 8, message: "Selector de calidad, audio o subtítulos abierto." },
      { action: "Simular fin de episodio", target: "S10", x: 24, y: 84, w: 26, h: 10 }
    ],
    "S09-episode-mobile": [
      { action: "Cerrar player", target: "S07-mobile", x: 2.5, y: 1.2, w: 18, h: 4.4 },
      { action: "Episodio anterior", target: "S09-episode-mobile", x: 12, y: 53.5, w: 28, h: 5.2, message: "Episodio anterior cargado." },
      { action: "Episodio siguiente", target: "S09-episode-mobile", x: 60, y: 53.5, w: 28, h: 5.2, message: "Episodio siguiente cargado." },
      { action: "Abrir calidad", target: "S09-episode-mobile", x: 4, y: 75.7, w: 92, h: 5.3, message: "Calidad abierta: Auto (720p)." },
      { action: "Abrir audio", target: "S09-episode-mobile", x: 4, y: 81, w: 92, h: 5.3, message: "Audio abierto: Español." },
      { action: "Simular fin de episodio", target: "S10-mobile", x: 35, y: 68, w: 30, h: 6 }
    ],
    "S06-remote": [
      { action: "Volver a Inicio", target: "S03", x: 9, y: 1.5, w: 7, h: 5.8 },
      { action: "Buscar torrent", target: "S11", x: 21.7, y: 44.2, w: 12.5, h: 6.2 },
      { action: "Actualizar metadatos", target: "S06-remote", x: 35, y: 44.2, w: 13, h: 6.2, message: "Metadatos actualizados; la disponibilidad no cambia." }
    ],
    "S06-pending": [
      { action: "Volver a Inicio", target: "S03", x: 9, y: 1.5, w: 7, h: 5.8 },
      { action: "Comprobar biblioteca", target: "S05", x: 21.7, y: 47.5, w: 34, h: 6.2 },
      { action: "Buscar otro torrent", target: "S11", x: 21.7, y: 54.2, w: 12.5, h: 6.2 },
      { action: "Actualizar metadatos", target: "S06-pending", x: 35, y: 54.2, w: 13, h: 6.2, message: "Metadatos actualizados; el torrent sigue pendiente." }
    ],
    "S06-plarr-down": [
      { action: "Volver a Inicio", target: "S03", x: 9, y: 1.5, w: 7, h: 5.8 },
      { action: "Reintentar Plarr", target: "S06", x: 60, y: 1.5, w: 10, h: 5.8, message: "Plarr reintentado: servicio sano." },
      { action: "Buscar torrent", target: "S11", x: 21.7, y: 45.8, w: 10.7, h: 6.2 },
      { action: "Actualizar metadatos", target: "S06-plarr-down", x: 32.8, y: 45.8, w: 13, h: 6.2, message: "Metadatos actualizados; Plarr sigue siendo un estado independiente." }
    ],
    "S06-torrents-down": [
      { action: "Volver a Inicio", target: "S03", x: 9, y: 1.5, w: 7, h: 5.8 },
      { action: "Reintentar qBittorrent", target: "S06", x: 76, y: 1.5, w: 12.5, h: 5.8, message: "Integraciones de torrent reintentadas." },
      { action: "Reanudar película", target: "S09", x: 21.7, y: 45.5, w: 10.3, h: 6.2 },
      { action: "Elegir descarga", target: "S06-torrents-down", x: 33, y: 45.5, w: 9.3, h: 6.2, message: "Las descargas siguen disponibles aunque qBittorrent no responda." },
      { action: "Actualizar metadatos", target: "S06-torrents-down", x: 42.8, y: 45.5, w: 13, h: 6.2, message: "Metadatos actualizados." }
    ]
  }
};

/* Round 3 turns every visual specimen into an individually navigable state. */
(() => {
  const prototype = window.NESTRR_PROTOTYPE;
  const hotspots = prototype.hotspots;
  const selfMessage = (action, target, x, y, w, h, message, tone) => ({ action, target, x, y, w, h, message, tone });

  Object.assign(hotspots, {
    S01: [{ action: "Crear administrador", target: "S01-loading", x: 54.5, y: 69, w: 36.5, h: 7 }],
    "S01-mobile": [{ action: "Crear administrador", target: "S01-loading-mobile", x: 10.5, y: 75.8, w: 79, h: 4.7 }],
    "S01-loading": [{ action: "Creación completada", target: "S03", x: 54.5, y: 69, w: 36.5, h: 7 }],
    "S01-loading-mobile": [{ action: "Creación completada", target: "S03-mobile", x: 10.5, y: 75.8, w: 79, h: 4.7 }],
    "S01-error": [{ action: "Ir a iniciar sesión", target: "S02", x: 77, y: 82, w: 14, h: 6 }],
    "S01-error-mobile": [{ action: "Ir a iniciar sesión", target: "S02-mobile", x: 63, y: 85, w: 28, h: 4.8 }],
    S02: [{ action: "Iniciar sesión", target: "S02-loading", x: 59.2, y: 56, w: 30, h: 7.6 }],
    "S02-mobile": [{ action: "Iniciar sesión", target: "S02-loading-mobile", x: 8, y: 57, w: 84, h: 7.2 }],
    "S02-loading": [{ action: "Acceso completado", target: "S03", x: 59.2, y: 56, w: 30, h: 7.6 }],
    "S02-loading-mobile": [{ action: "Acceso completado", target: "S03-mobile", x: 8, y: 57, w: 84, h: 7.2 }],
    "S02-error": [{ action: "Reintentar", target: "S02-loading", x: 59.2, y: 60, w: 30, h: 7.6 }],
    "S02-error-mobile": [{ action: "Reintentar", target: "S02-loading-mobile", x: 8, y: 61, w: 84, h: 7.2 }],

    S04: [
      { action: "Nueva búsqueda", target: "S04-loading", x: 80.5, y: 22.7, w: 16.7, h: 7.5 },
      { action: "Abrir Órbita Ciega", target: "S06-remote", x: 2.8, y: 46.5, w: 14.8, h: 40 },
      { action: "Abrir Puerto Umbral", target: "S07", x: 49.8, y: 46.5, w: 14.5, h: 40 }
    ],
    "S04-mobile": [
      { action: "Nueva búsqueda", target: "S04-loading-mobile", x: 4.5, y: 18.8, w: 91, h: 4.2 },
      { action: "Abrir Órbita Ciega", target: "S06-remote-mobile", x: 4.5, y: 32.5, w: 45, h: 20.5 },
      { action: "Abrir Puerto Umbral", target: "S07-mobile", x: 50.5, y: 32.5, w: 45, h: 20.5 },
      { action: "Ir a Inicio", target: "S03-mobile", x: 4.5, y: 93.5, w: 29, h: 5 },
      { action: "Ir a Descargado", target: "S05-mobile", x: 65.5, y: 93.5, w: 30, h: 5 }
    ],
    "S04-ready": [{ action: "Buscar", target: "S04-loading", x: 80.5, y: 22.7, w: 16.7, h: 7.5 }],
    "S04-ready-mobile": [{ action: "Buscar", target: "S04-loading-mobile", x: 4.5, y: 18.8, w: 91, h: 4.2 }],
    "S04-loading": [{ action: "Resultados recibidos", target: "S04", x: 3, y: 38, w: 24, h: 7 }],
    "S04-loading-mobile": [{ action: "Resultados recibidos", target: "S04-mobile", x: 4.5, y: 29, w: 70, h: 5 }],
    "S04-empty": [{ action: "Cambiar búsqueda", target: "S04-ready", x: 3, y: 22.7, w: 77, h: 7.5 }],
    "S04-empty-mobile": [{ action: "Cambiar búsqueda", target: "S04-ready-mobile", x: 4.5, y: 13, w: 91, h: 5 }],
    "S04-error": [{ action: "Reintentar", target: "S04-loading", x: 43, y: 54, w: 14, h: 7, tone: "alert" }],
    "S04-error-mobile": [{ action: "Reintentar", target: "S04-loading-mobile", x: 15, y: 49, w: 70, h: 5, tone: "alert" }],

    "S06-remote": [
      { action: "Volver a Inicio", target: "S03", x: 9, y: 1.5, w: 7, h: 5.8 },
      { action: "Buscar torrent", target: "S11-ready", x: 21.7, y: 44.2, w: 12.5, h: 6.2 }
    ],
    "S06-remote-mobile": [
      { action: "Volver a Inicio", target: "S03-mobile", x: 3, y: 1, w: 16, h: 4.5 },
      { action: "Buscar torrent", target: "S11-ready-mobile", x: 6, y: 45, w: 43, h: 5 }
    ],
    "S06-pending": [
      { action: "Ver biblioteca", target: "S05", x: 21.7, y: 47.5, w: 34, h: 6.2 },
      selfMessage("Actualizar", "S06-pending", 35, 54.2, 13, 6.2, "Estado actualizado: el archivo todavía no está disponible.")
    ],
    "S06-pending-mobile": [
      { action: "Ver biblioteca", target: "S05-mobile", x: 6, y: 49, w: 88, h: 5 },
      { action: "Volver a Inicio", target: "S03-mobile", x: 3, y: 1, w: 16, h: 4.5 }
    ],
    "S06-plarr-down": [
      { action: "Recuperar Plarr", target: "S06", x: 57, y: 1.5, w: 13, h: 5.8 },
      { action: "Buscar torrent", target: "S11-ready", x: 21.7, y: 45.8, w: 10.7, h: 6.2 }
    ],
    "S06-jackett-down": [
      { action: "Reanudar", target: "S09", x: 21.7, y: 45.5, w: 10.5, h: 6.2 },
      { action: "Recuperar Jackett", target: "S06", x: 68, y: 1.5, w: 13, h: 5.8 }
    ],
    "S06-jackett-down-mobile": [
      { action: "Reanudar", target: "S09-mobile", x: 5.5, y: 42.4, w: 89, h: 5.1 },
      { action: "Recuperar Jackett", target: "S06-mobile", x: 35, y: 4.5, w: 45, h: 4.8 }
    ],
    "S06-qbittorrent-down": [
      { action: "Reanudar", target: "S09", x: 21.7, y: 45.5, w: 10.3, h: 6.2 },
      { action: "Recuperar qBittorrent", target: "S06", x: 76, y: 1.5, w: 12.5, h: 5.8 }
    ],

    S07: [
      { action: "Volver a Inicio", target: "S03", x: 9.8, y: 1.5, w: 6.4, h: 5.7 },
      { action: "Buscar torrent de la serie", target: "S11-ready", x: 17.7, y: 44.4, w: 11.5, h: 5.8 },
      selfMessage("Alternar Temporada 1", "S07", 3, 51, 94, 5.6, "Temporada 1 contraída o expandida."),
      { action: "Reproducir S01E01", target: "S09-episode-e01", x: 81.5, y: 57, w: 11.8, h: 6 },
      { action: "Reanudar S01E02", target: "S09-episode", x: 81.5, y: 64.2, w: 11.8, h: 6 },
      { action: "Buscar torrent S01E03", target: "S11-ready", x: 81.5, y: 71.7, w: 11.8, h: 6 },
      selfMessage("Reintentar Temporada 2", "S07", 43, 92.5, 18.5, 5.3, "Temporada 2 reintentada de forma localizada.", "alert")
    ],
    "S07-mobile": [
      { action: "Buscar torrent de la serie", target: "S11-ready-mobile", x: 5.3, y: 39.1, w: 40, h: 4.2 },
      selfMessage("Alternar temporada", "S07-mobile", 3.4, 44, 93, 4.8, "Temporada 1 contraída o expandida."),
      { action: "Reproducir S01E01", target: "S09-episode-e01-mobile", x: 6, y: 57.4, w: 88, h: 3.7 },
      { action: "Reanudar S01E02", target: "S09-episode-mobile", x: 6, y: 70.2, w: 88, h: 3.7 },
      { action: "Buscar torrent S01E03", target: "S11-ready-mobile", x: 6, y: 83.6, w: 88, h: 3.7 },
      { action: "Volver a Inicio", target: "S03-mobile", x: 1, y: 94.6, w: 16, h: 5.3 }
    ],

    S08: [
      { action: "Abrir título", target: "S06", x: 2.8, y: 50.8, w: 30.9, h: 21 },
      { action: "Volver", target: "S03", x: 11, y: 1.4, w: 6.7, h: 6 }
    ],
    "S08-mobile": [
      { action: "Abrir título", target: "S06-mobile", x: 5, y: 38, w: 90, h: 18 },
      { action: "Volver", target: "S03-mobile", x: 4, y: 94, w: 29, h: 5 }
    ],
    "S08-empty": [{ action: "Explorar", target: "S04-ready", x: 40, y: 55, w: 20, h: 7 }],
    "S08-empty-mobile": [{ action: "Explorar", target: "S04-ready-mobile", x: 15, y: 52, w: 70, h: 5 }],
    "S08-error": [{ action: "Reintentar", target: "S08", x: 40, y: 55, w: 20, h: 7, tone: "alert" }],
    "S08-error-mobile": [{ action: "Reintentar", target: "S08-mobile", x: 15, y: 52, w: 70, h: 5, tone: "alert" }],

    "S09-episode": [
      { action: "Cerrar", target: "S07", x: 2, y: 2, w: 8, h: 5.2 },
      { action: "Anterior E01", target: "S09-episode-e01", x: 38.5, y: 72.5, w: 10, h: 6 },
      { action: "E03 no disponible", target: "S07", x: 52, y: 72.5, w: 10, h: 6 }
    ],
    "S09-episode-mobile": [
      { action: "Cerrar", target: "S07-mobile", x: 2.5, y: 1.2, w: 18, h: 4.4 },
      { action: "Anterior E01", target: "S09-episode-e01-mobile", x: 12, y: 53.5, w: 28, h: 5.2 },
      { action: "E03 no disponible", target: "S07-mobile", x: 60, y: 53.5, w: 28, h: 5.2 }
    ],
    "S09-episode-e01": [
      { action: "Cerrar", target: "S07", x: 2, y: 2, w: 8, h: 5.2 },
      { action: "Siguiente E02", target: "S09-episode", x: 52, y: 72.5, w: 10, h: 6 }
    ],
    "S09-episode-e01-mobile": [
      { action: "Cerrar", target: "S07-mobile", x: 2.5, y: 1.2, w: 18, h: 4.4 },
      { action: "Siguiente E02", target: "S09-episode-mobile", x: 60, y: 53.5, w: 28, h: 5.2 }
    ],
    "S09-episode-e05": [
      { action: "Finalizar E05", target: "S10", x: 30, y: 82, w: 45, h: 12 },
      { action: "Cerrar", target: "S07", x: 2, y: 2, w: 8, h: 5.2 }
    ],
    "S09-episode-e05-mobile": [
      { action: "Finalizar E05", target: "S10-mobile", x: 4, y: 68, w: 92, h: 6 },
      { action: "Cerrar", target: "S07-mobile", x: 2.5, y: 1.2, w: 18, h: 4.4 }
    ],
    "S09-episode-e06": [
      { action: "Anterior E05", target: "S09-episode-e05", x: 38.5, y: 72.5, w: 10, h: 6 },
      { action: "Cerrar", target: "S07", x: 2, y: 2, w: 8, h: 5.2 }
    ],
    "S09-episode-e06-mobile": [
      { action: "Anterior E05", target: "S09-episode-e05-mobile", x: 12, y: 53.5, w: 28, h: 5.2 },
      { action: "Cerrar", target: "S07-mobile", x: 2.5, y: 1.2, w: 18, h: 4.4 }
    ],

    S10: [
      { action: "Cerrar player", target: "S07", x: 2, y: 3.2, w: 7.2, h: 5.6 },
      { action: "Reproducir ahora", target: "S09-episode-e06", x: 65.5, y: 62.3, w: 27.6, h: 6.2 },
      { action: "Cancelar autoplay", target: "S09-episode-e05", x: 65.5, y: 68.6, w: 27.6, h: 5.7 },
      { action: "Ir al episodio anterior", target: "S09-episode-e05", x: 39.7, y: 90.1, w: 8.5, h: 5.8 }
    ],
    "S10-mobile": [
      { action: "Cerrar player", target: "S07-mobile", x: 3.5, y: 1.3, w: 19, h: 4.2 },
      { action: "Reproducir ahora", target: "S09-episode-e06-mobile", x: 7, y: 77, w: 86, h: 4.5 },
      { action: "Cancelar autoplay", target: "S09-episode-e05-mobile", x: 7, y: 82.1, w: 86, h: 4.5 },
      { action: "Ir al episodio anterior", target: "S09-episode-e05-mobile", x: 3.5, y: 91, w: 25, h: 5 }
    ],

    S11: [
      { action: "Enviar torrent", target: "S11-sending", x: 69.8, y: 57, w: 8.5, h: 5 },
      { action: "Volver", target: "S06-remote", x: 77.5, y: 9.5, w: 3.2, h: 5 }
    ],
    "S11-mobile": [
      { action: "Enviar torrent", target: "S11-sending-mobile", x: 9, y: 52.2, w: 82, h: 4.2 },
      { action: "Volver", target: "S06-remote-mobile", x: 85.5, y: 2.5, w: 10, h: 4.4 }
    ],
    "S11-ready": [{ action: "Buscar", target: "S11-loading", x: 56.4, y: 25.2, w: 12, h: 6.6 }],
    "S11-ready-mobile": [{ action: "Buscar", target: "S11-loading-mobile", x: 7, y: 24, w: 86, h: 5.4 }],
    "S11-loading": [{ action: "Resultados recibidos", target: "S11", x: 40, y: 45, w: 20, h: 8 }],
    "S11-loading-mobile": [{ action: "Resultados recibidos", target: "S11-mobile", x: 15, y: 43, w: 70, h: 6 }],
    "S11-sending": [{ action: "Envío aceptado", target: "S11-accepted", x: 69.8, y: 57, w: 8.5, h: 5 }],
    "S11-sending-mobile": [{ action: "Envío aceptado", target: "S11-accepted-mobile", x: 9, y: 52.2, w: 82, h: 4.2 }],
    "S11-accepted": [{ action: "Ver película", target: "S06-pending", x: 60, y: 18, w: 18, h: 5.5 }],
    "S11-accepted-mobile": [{ action: "Ver película", target: "S06-pending-mobile", x: 8, y: 31, w: 84, h: 5 }],
    "S11-error": [{ action: "Reintentar", target: "S11", x: 69.8, y: 57, w: 8.5, h: 5, tone: "alert" }],
    "S11-error-mobile": [{ action: "Reintentar", target: "S11-mobile", x: 9, y: 52.2, w: 82, h: 4.2, tone: "alert" }],

    S12: [
      { action: "Guardar", target: "S12-saved", x: 4.8, y: 55, w: 40, h: 6.4 },
      { action: "Eliminar perfil", target: "S12-delete-dialog", x: 4.8, y: 77, w: 22, h: 6.2 },
      { action: "Abrir menú", target: "S12-menu", x: 94, y: 1.2, w: 4.5, h: 6.5 }
    ],
    "S12-mobile": [
      { action: "Guardar", target: "S12-saved-mobile", x: 8, y: 42, w: 84, h: 5.5 },
      { action: "Eliminar perfil", target: "S12-delete-dialog-mobile", x: 8, y: 68, w: 48, h: 5.5 },
      { action: "Abrir menú", target: "S12-menu-mobile", x: 86, y: 1.5, w: 10, h: 4.5 }
    ],
    "S12-saved": [{ action: "Volver a inicio", target: "S03", x: 11, y: 1.4, w: 6.7, h: 6 }],
    "S12-saved-mobile": [{ action: "Volver a inicio", target: "S03-mobile", x: 4, y: 94, w: 29, h: 5 }],
    "S12-delete-dialog": [
      { action: "Cancelar", target: "S12", x: 70.2, y: 75.9, w: 7.2, h: 6.2 },
      { action: "Confirmar borrado", target: "S02", x: 78.5, y: 75.9, w: 13.3, h: 6.2 }
    ],
    "S12-delete-dialog-mobile": [
      { action: "Cancelar", target: "S12-mobile", x: 11, y: 84, w: 38, h: 4.5 },
      { action: "Confirmar borrado", target: "S02-mobile", x: 51, y: 84, w: 37, h: 4.5 }
    ],
    "S12-menu": [
      { action: "Administración", target: "S13", x: 84.5, y: 24.6, w: 11.5, h: 5.2 },
      { action: "Cerrar menú", target: "S12", x: 60, y: 20, w: 20, h: 35 }
    ],
    "S12-menu-mobile": [
      { action: "Administración", target: "S13-mobile", x: 60, y: 13.5, w: 35, h: 4.5 },
      { action: "Cerrar menú", target: "S12-mobile", x: 5, y: 12, w: 45, h: 30 }
    ],

    S13: [
      { action: "Crear duplicado", target: "S13-duplicate-error", x: 46.5, y: 36.2, w: 10.2, h: 5.1 },
      { action: "Abrir miembro", target: "S13-member-menu", x: 53.8, y: 55.8, w: 3, h: 5 },
      { action: "Restablecer contraseña", target: "S13-reset-dialog", x: 38, y: 56, w: 15, h: 5 }
    ],
    "S13-mobile": [
      { action: "Crear duplicado", target: "S13-duplicate-error-mobile", x: 7.5, y: 37.5, w: 85, h: 4.2 },
      { action: "Abrir miembro", target: "S13-member-menu-mobile", x: 82, y: 55, w: 10, h: 4.5 },
      { action: "Restablecer contraseña", target: "S13-reset-dialog-mobile", x: 48, y: 55, w: 33, h: 4.5 }
    ],
    "S13-duplicate-error": [{ action: "Corregir usuario", target: "S13", x: 3, y: 23, w: 40, h: 6 }],
    "S13-duplicate-error-mobile": [{ action: "Corregir usuario", target: "S13-mobile", x: 8, y: 18, w: 84, h: 5 }],
    "S13-member-menu": [
      { action: "Restablecer contraseña", target: "S13-reset-dialog", x: 33, y: 62, w: 16, h: 5.2 },
      { action: "Ver como Bruno", target: "S03", x: 33, y: 68, w: 16, h: 5.2, setFlag: "impersonating" },
      { action: "Cerrar menú", target: "S13", x: 60, y: 45, w: 25, h: 30 }
    ],
    "S13-member-menu-mobile": [
      { action: "Restablecer contraseña", target: "S13-reset-dialog-mobile", x: 48, y: 66, w: 35, h: 4.5 },
      { action: "Ver como Bruno", target: "S03-mobile", x: 48, y: 61, w: 35, h: 4.5, setFlag: "impersonating" },
      { action: "Cerrar menú", target: "S13-mobile", x: 5, y: 50, w: 35, h: 25 }
    ],
    "S13-reset-dialog": [
      { action: "Cancelar", target: "S13", x: 45, y: 82, w: 10, h: 5.2 },
      { action: "Confirmar", target: "S13", x: 56, y: 82, w: 13, h: 5.2 }
    ],
    "S13-reset-dialog-mobile": [
      { action: "Cancelar", target: "S13-mobile", x: 42, y: 78, w: 18, h: 4.5 },
      { action: "Confirmar", target: "S13-mobile", x: 62, y: 78, w: 24, h: 4.5 }
    ]
  });

  hotspots["S03-mobile"].push({ action: "Abrir ajustes", target: "S12-menu-mobile", x: 86, y: 1.2, w: 10, h: 4.8 });
})();
