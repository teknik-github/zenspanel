package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zenspanel/zenspanel/internal/auth"
	"github.com/zenspanel/zenspanel/internal/store"
)

type PackageHandler struct {
	packages *store.PackageStore
}

func NewPackageHandler(packages *store.PackageStore) *PackageHandler {
	return &PackageHandler{packages: packages}
}

// packageRequest accepts disk_quota and memory_limit in MB (V49).
// The UI sends MB; we convert to bytes before storing.
type packageRequest struct {
	Name               string `json:"name"`
	CPUQuota           int    `json:"cpu_quota"`
	DiskQuotaMB        int64  `json:"disk_quota_mb"`
	MemoryLimitMB      int64  `json:"memory_limit_mb"`
	MaxDomains         int    `json:"max_domains"`
	MaxDatabases       int    `json:"max_databases"`
	MaxCronJobs        int    `json:"max_cron_jobs"`
	MaxProcs           int    `json:"max_procs"`
	IOReadMbps         int64  `json:"io_read_mbps"`
	IOWriteMbps        int64  `json:"io_write_mbps"`
	AntivirusEnabled   bool   `json:"antivirus_enabled"`
	MaxFTPAccounts     int    `json:"max_ftp_accounts"`
	PHPVersionsAllowed string `json:"php_versions_allowed"`
	TerminalEnabled    bool   `json:"terminal_enabled"`
	BackupEnabled      bool   `json:"backup_enabled"`
}

func (r packageRequest) toPackage() store.Package {
	return store.Package{
		Name:               r.Name,
		CPUQuota:           r.CPUQuota,
		DiskQuota:          r.DiskQuotaMB * 1024 * 1024,
		MemoryLimit:        r.MemoryLimitMB * 1024 * 1024,
		MaxDomains:         r.MaxDomains,
		MaxDatabases:       r.MaxDatabases,
		MaxCronJobs:        r.MaxCronJobs,
		MaxProcs:           r.MaxProcs,
		IOReadBps:          r.IOReadMbps * 1024 * 1024,
		IOWriteBps:         r.IOWriteMbps * 1024 * 1024,
		AntivirusEnabled:   r.AntivirusEnabled,
		MaxFTPAccounts:     r.MaxFTPAccounts,
		PHPVersionsAllowed: r.PHPVersionsAllowed,
		TerminalEnabled:    r.TerminalEnabled,
		BackupEnabled:      r.BackupEnabled,
	}
}

// packageAdminResponse includes all fields for admin use.
func packageAdminResponse(p store.Package) map[string]interface{} {
	return map[string]interface{}{
		"id":                   p.ID,
		"name":                 p.Name,
		"cpu_quota":            p.CPUQuota,
		"disk_quota":           p.DiskQuota,
		"disk_quota_mb":        p.DiskQuota / (1024 * 1024),
		"memory_limit":         p.MemoryLimit,
		"memory_limit_mb":      p.MemoryLimit / (1024 * 1024),
		"max_domains":          p.MaxDomains,
		"max_databases":        p.MaxDatabases,
		"max_cron_jobs":        p.MaxCronJobs,
		"max_procs":            p.MaxProcs,
		"io_read_bps":          p.IOReadBps,
		"io_read_mbps":         p.IOReadBps / (1024 * 1024),
		"io_write_bps":         p.IOWriteBps,
		"io_write_mbps":        p.IOWriteBps / (1024 * 1024),
		"antivirus_enabled":    p.AntivirusEnabled,
		"max_ftp_accounts":     p.MaxFTPAccounts,
		"php_versions_allowed": p.PHPVersionsAllowed,
		"terminal_enabled":     p.TerminalEnabled,
		"backup_enabled":       p.BackupEnabled,
		"created_at":           p.CreatedAt,
		"updated_at":           p.UpdatedAt,
	}
}

// packageUserResponse returns only customer-visible quota fields.
// Internal system limits (max_procs, io_*_bps, cpu_quota) are omitted.
func packageUserResponse(p store.Package) map[string]interface{} {
	return map[string]interface{}{
		"id":                   p.ID,
		"name":                 p.Name,
		"disk_quota_mb":        p.DiskQuota / (1024 * 1024),
		"memory_limit_mb":      p.MemoryLimit / (1024 * 1024),
		"max_domains":          p.MaxDomains,
		"max_databases":        p.MaxDatabases,
		"max_cron_jobs":        p.MaxCronJobs,
		"max_ftp_accounts":     p.MaxFTPAccounts,
		"antivirus_enabled":    p.AntivirusEnabled,
		"php_versions_allowed": p.PHPVersionsAllowed,
		"terminal_enabled":     p.TerminalEnabled,
		"backup_enabled":       p.BackupEnabled,
	}
}

// packageResponse is an alias for admin use.
func packageResponse(p store.Package) map[string]interface{} {
	return packageAdminResponse(p)
}

func (h *PackageHandler) List(c *gin.Context) {
	pkgs, err := h.packages.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	isAdmin := auth.GetRole(c) == "admin"
	resp := make([]map[string]interface{}, len(pkgs))
	for i, p := range pkgs {
		if isAdmin {
			resp[i] = packageAdminResponse(p)
		} else {
			resp[i] = packageUserResponse(p)
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func (h *PackageHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	pkg, err := h.packages.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "package not found"})
		return
	}
	if auth.GetRole(c) == "admin" {
		c.JSON(http.StatusOK, packageAdminResponse(*pkg))
	} else {
		c.JSON(http.StatusOK, packageUserResponse(*pkg))
	}
}

func (h *PackageHandler) Create(c *gin.Context) {
	var req packageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pkg := req.toPackage()
	if err := h.packages.Create(&pkg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, packageAdminResponse(pkg))
}

func (h *PackageHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req packageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pkg := req.toPackage()
	if err := h.packages.Update(id, &pkg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (h *PackageHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.packages.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
