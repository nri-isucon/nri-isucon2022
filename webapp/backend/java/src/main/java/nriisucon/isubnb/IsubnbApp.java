package nriisucon.isubnb;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.nio.charset.StandardCharsets;
import java.sql.Connection;
import java.sql.SQLException;
import java.time.LocalDate;
import java.time.format.DateTimeFormatter;
import java.time.temporal.ChronoUnit;
import java.util.ArrayList;
import java.util.Base64;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import java.util.regex.Matcher;
import java.util.regex.Pattern;
import java.util.stream.Collectors;

import nriisucon.isubnb.model.*;
import nriisucon.isubnb.response.*;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.core.io.Resource;
import org.springframework.core.io.ResourceLoader;
import org.springframework.dao.EmptyResultDataAccessException;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpStatus;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.jdbc.core.BeanPropertyRowMapper;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.jdbc.datasource.DataSourceUtils;
import org.springframework.jdbc.datasource.init.ResourceDatabasePopulator;
import org.springframework.web.bind.annotation.CrossOrigin;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.multipart.MultipartFile;

import nriisucon.isubnb.error.ClientException;
import nriisucon.isubnb.request.InitializeRequest;
import nriisucon.isubnb.request.ReserveActivityRequest;
import nriisucon.isubnb.request.ReserveHomeRequest;

@SpringBootApplication
@RestController
@CrossOrigin
@RequestMapping("/api/v1")
public class IsubnbApp {

	@Autowired
	private JdbcTemplate db;

	@Autowired
	protected ResourceLoader resourceLoader;

	@PostMapping("/initialize")
	public String initialize(@RequestBody InitializeRequest initReq) throws SQLException {
		executeScript("sql/0_Schema.sql");
		executeScript("sql/1_CsvDataImport.sql");
		db.update("INSERT INTO isubnb.config VALUES (?)", initReq.getReservableDays());
		return "{\"language\" : \"java\"}";
	}

	@GetMapping("/homes")
	public HomesResponse getHomes(
			@RequestParam(value = "location", required = false) String location,
			@RequestParam(value = "start_date", required = false) String startDate,
			@RequestParam(value = "end_date", required = false) String endDate,
			@RequestParam(value = "number_of_people", required = false) Integer numberOfPeople,
			@RequestParam(value = "style", required = false) String style) {

		List<Home> homes = db.query("SELECT * FROM isubnb.home ORDER BY rate DESC, price ASC, name ASC",
				new BeanPropertyRowMapper<Home>(Home.class));

		if (startDate != null && endDate != null) {
			List<Home> filteredHomes = new ArrayList<Home>();
			for (Home home : homes) {
				List<ReserveHome> reservationHomes = db.query(
						"SELECT * FROM isubnb.reservation_home WHERE home_id = ? AND ? <= date AND date < ?",
						new BeanPropertyRowMapper<ReserveHome>(ReserveHome.class), home.getId(), startDate, endDate);
				if (reservationHomes.size() == 0) {
					filteredHomes.add(home);
				}
			}
			homes = filteredHomes;
		}
		if (location != null) {
			homes = homes.stream().filter(x -> x.getLocation().equals(location))
					.collect(Collectors.toList());
		}
		if (style != null) {
			homes = homes.stream().filter(x -> x.getStyle().equals(style))
					.collect(Collectors.toList());
		}
		if (numberOfPeople != null && numberOfPeople > 0) {
			homes = homes.stream().filter(x -> x.getMaxPeopleNum() >= numberOfPeople)
					.collect(Collectors.toList());
		}

		homes = homes.stream().map(x -> convertToResponseHome(x)).collect(Collectors.toList());

		return new HomesResponse(homes);
	}

	@GetMapping("/home/{homeId}")
	public Home getHome(@PathVariable("homeId") String homeId) {

		try {
			db.queryForObject("SELECT * FROM isubnb.home WHERE id = ?",
					new BeanPropertyRowMapper<Home>(Home.class), homeId);
		} catch (EmptyResultDataAccessException e) {
			throw new ClientException("対象宿が存在しません。");
		}

		Home home = db.queryForObject("SELECT * FROM isubnb.home WHERE id = ?",
				new BeanPropertyRowMapper<Home>(Home.class), homeId);
		convertToResponseHome(home);

		return home;
	}

	@GetMapping(value = "/home/{homeId}/image/{imageId}", produces = MediaType.IMAGE_JPEG_VALUE)
	public byte[] getHomeImage(@PathVariable String homeId, @PathVariable String imageId) {

		try {
			db.queryForObject("SELECT * FROM isubnb.home WHERE id = ?",
					new BeanPropertyRowMapper<Home>(Home.class), homeId);
		} catch (EmptyResultDataAccessException e) {
			throw new ClientException("対象宿が存在しません。");
		}

		Home home = db.queryForObject("SELECT * FROM isubnb.home WHERE id = ?",
				new BeanPropertyRowMapper<Home>(Home.class), homeId);

		Map<String, String> result = new HashMap<>();
		result.put("photo_1", home.getPhoto_1());
		result.put("photo_2", home.getPhoto_2());
		result.put("photo_3", home.getPhoto_3());
		result.put("photo_4", home.getPhoto_4());
		result.put("photo_5", home.getPhoto_5());

		if (Integer.valueOf(imageId) < 1 || Integer.valueOf(imageId) > 5) {
			throw new ClientException("画像IDの指定が誤っています。");
		}
		if (result.get("photo_" + imageId) == null) {
			throw new ClientException("画像が存在しません。");
		}

		return Base64.getDecoder().decode(result.get("photo_" + imageId));
	}

	@GetMapping("/home/{homeId}/calendar")
	public CalendarResponse getHomeCalendar(@PathVariable String homeId) {

		int reservableDays = db.queryForObject(
				"SELECT reservable_days FROM isubnb.config", Integer.class);
		if (reservableDays == 0) {
			throw new ClientException("予約可能日数が0日です。");
		}

		List<Home> targetHomes = db.query("SELECT * FROM isubnb.home WHERE id = ?",
				new BeanPropertyRowMapper<Home>(Home.class), homeId);
		if (targetHomes == null || targetHomes.size() == 0) {
			throw new ClientException("対象宿が存在しません。");
		}

		DateTimeFormatter formatter = DateTimeFormatter.ISO_LOCAL_DATE;
		LocalDate date = LocalDate.now();
		LocalDate endDate = date.plusDays(reservableDays);

		List<CalendarDetail> calendarList = new ArrayList<>();
		while (ChronoUnit.DAYS.between(date, endDate) >= 1) {
			List<Home> targetHomeReservation = db.query(
					"SELECT * FROM isubnb.reservation_home WHERE home_id = ? AND DATE(date) = ? AND is_deleted = ? ORDER BY user_id, home_id",
					new BeanPropertyRowMapper<Home>(Home.class), homeId, date.format(formatter), 0);
			boolean isAvailable = (targetHomeReservation.size() > 0) ? false : true;
			calendarList.add(new CalendarDetail(date, isAvailable));
			date = date.plusDays(1);
		}

		return new CalendarResponse(calendarList);
	}

	@PostMapping("/reservation_home")
	public String createHomeReservation(@RequestBody ReserveHomeRequest req) {

		if (req.getUserId() == null) {
			throw new ClientException("ユーザIDを入力してください。");
		}
		try {
			db.queryForObject("SELECT * FROM isubnb.user WHERE id = ?",
					new BeanPropertyRowMapper<User>(User.class), req.getUserId());
		} catch (EmptyResultDataAccessException e) {
			throw new ClientException("対象ユーザが存在しません。");
		}

		if (req.getHomeId() == null) {
			throw new ClientException("宿IDを入力してください。");
		}
		try {
			db.queryForObject("SELECT * FROM isubnb.home WHERE id = ?",
					new BeanPropertyRowMapper<Home>(Home.class), req.getHomeId());
		} catch (EmptyResultDataAccessException e) {
			throw new ClientException("対象宿が存在しません。");
		}

		Home home = db.queryForObject("SELECT * FROM isubnb.home WHERE id = ?",
				new BeanPropertyRowMapper<Home>(Home.class), req.getHomeId());

		Matcher matcherNum = Pattern.compile("[0-9]+").matcher(String.valueOf(req.getNumberOfPeople()));
		if (!matcherNum.matches()) {
			throw new ClientException("人数は数値で入力してください。");
		}

		if (!(req.getNumberOfPeople() <= home.getMaxPeopleNum())) {
			throw new ClientException("予約可能人数を超えています。");
		}

		Matcher matcherStartDate = Pattern.compile("^[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])$")
				.matcher(req.getStartDate());
		Matcher matcherEndDate = Pattern.compile("^[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])$")
				.matcher(req.getEndDate());
		if (!matcherStartDate.matches() || !matcherEndDate.matches()) {
			throw new ClientException("日付はyyyy-mm-dd形式で入力してください。");
		}

		DateTimeFormatter formatter = DateTimeFormatter.ISO_LOCAL_DATE;
		LocalDate startDate = LocalDate.parse(req.getStartDate(), formatter);
		LocalDate endDate = LocalDate.parse(req.getEndDate(), formatter);
		if (ChronoUnit.DAYS.between(startDate, endDate) < 1) {
			throw new ClientException("日付間隔を1日以上にしてください。");
		}

		List<Map<String, Object>> reserved_home_list = db.queryForList(
				"SELECT * FROM isubnb.home h JOIN isubnb.reservation_home rh ON h.id=rh.home_id " +
						"WHERE h.id = ? " +
						"AND ? <= rh.date " +
						"AND rh.date < ?" +
						"AND rh.is_deleted = ?",
				req.getHomeId(), req.getStartDate(), req.getEndDate(), 0);
		if (reserved_home_list.size() != 0) {
			throw new ClientException("既に予約が入っているため、予約できません。");
		}

		String reserve_id = String.valueOf(UUID.randomUUID());
		LocalDate date = startDate;
		while (ChronoUnit.DAYS.between(date, endDate) >= 1) {
			db.update("INSERT INTO isubnb.reservation_home(id, user_id, home_id, date, " +
					"number_of_people, is_deleted) VALUES (?, ?, ?, ?, ?, false)",
					reserve_id, req.getUserId(), req.getHomeId(), date, req.getNumberOfPeople());
			date = date.plusDays(1);
		}

		return "{\"result\" : true}";
	}

	@GetMapping("/user/{userId}/reservation_home")
	public ReservationHomeResponse getHomeReservations(@PathVariable String userId) {

		try {
			db.queryForObject("SELECT * FROM isubnb.user WHERE id = ?",
					new BeanPropertyRowMapper<User>(User.class), userId);
		} catch (EmptyResultDataAccessException e) {
			throw new ClientException("対象ユーザが存在しません。");
		}

		List<Map<String, Object>> getList = db.queryForList(
				"SELECT DISTINCT rh.id as reservation_id, rh.number_of_people, rh.home_id " +
						"FROM isubnb.user u " +
						"JOIN isubnb.reservation_home rh ON u.id = rh.user_id " +
						"WHERE u.id = ? " +
						"AND rh.is_deleted = ?",
				userId, 0);

		List<ReserveHomeDetail> resultList = new ArrayList<>();
		for (Map<String, Object> map : getList) {
			Home home = db.queryForObject("SELECT * FROM isubnb.home WHERE id = ?",
					new BeanPropertyRowMapper<Home>(Home.class), map.get("home_id"));
			convertToResponseHome(home);

			String reserveId = (String) map.get("reservation_id");
			int numberOfPeople = (int) map.get("number_of_people");

			LocalDate startDate = db.queryForObject(
					"SELECT min(date) FROM isubnb.reservation_home WHERE id = '" + reserveId + "'", LocalDate.class);

			LocalDate endDate = db.queryForObject(
					"SELECT max(date) FROM isubnb.reservation_home WHERE id = '" + reserveId + "'", LocalDate.class);
			endDate = endDate.plusDays(1);

			resultList.add(new ReserveHomeDetail(reserveId, startDate, endDate, numberOfPeople, home));

		}
		return new ReservationHomeResponse(resultList);
	}

	@DeleteMapping("/reservation_home/{reserveId}")
	public String deleteHomeReservation(@PathVariable String reserveId) {
		List<ReserveHome> reservationHomes = db.query(
				"SELECT * FROM isubnb.reservation_home WHERE id = ? AND is_deleted = ?",
				new BeanPropertyRowMapper<ReserveHome>(ReserveHome.class), reserveId, 0);
		if (reservationHomes.size() == 0) {
			throw new ClientException("対象の予約が存在しませんでした。");
		}
		db.update("UPDATE isubnb.reservation_home SET is_deleted = ? WHERE id = ?", 1, reserveId);
		return "{\"result\" : true}";
	}

	@GetMapping("/activities")
	public ActivityResponse getActivities(
			@RequestParam(value = "location", required = false) String location,
			@RequestParam(value = "date", required = false) String date) {

		List<Activity> activities = db.query("SELECT * FROM isubnb.activity ORDER BY rate DESC, price ASC, name ASC",
				new BeanPropertyRowMapper<Activity>(Activity.class));

		if (location != null) {
			activities = activities.stream().filter(x -> x.getLocation().equals(location))
					.collect(Collectors.toList());
		}
		if (date != null) {
			List<Activity> filteredActivity = new ArrayList<Activity>();
			for (Activity activity : activities) {
				List<ReserveActivity> reservationActivities = db.query(
						"SELECT * FROM isubnb.reservation_activity WHERE activity_id = ? AND date = ?",
						new BeanPropertyRowMapper<ReserveActivity>(ReserveActivity.class), activity.getId(), date);
				if (reservationActivities.size() == 0) {
					filteredActivity.add(activity);
				}
			}
			activities = filteredActivity;
		}

		activities = activities.stream().map(x -> convertToResponseActivity(x)).collect(Collectors.toList());

		return new ActivityResponse(activities);
	}

	@GetMapping("/activity/{activityId}")
	public Activity getActivity(@PathVariable String activityId) {

		try {
			db.queryForObject("SELECT * FROM isubnb.activity WHERE id = ?",
					new BeanPropertyRowMapper<Activity>(Activity.class), activityId);
		} catch (EmptyResultDataAccessException e) {
			throw new ClientException("対象アクティビティが存在しません。");
		}

		Activity activity = db.queryForObject("SELECT * FROM isubnb.activity WHERE id = ?",
				new BeanPropertyRowMapper<Activity>(Activity.class), activityId);
		convertToResponseActivity(activity);

		return activity;
	}

	@GetMapping(value = "/activity/{activityId}/image/{imageId}", produces = MediaType.IMAGE_JPEG_VALUE)
	public byte[] getActivityImage(@PathVariable int activityId, @PathVariable int imageId) {

		try {
			db.queryForObject("SELECT * FROM isubnb.activity WHERE id = ?",
					new BeanPropertyRowMapper<Activity>(Activity.class), activityId);
		} catch (EmptyResultDataAccessException e) {
			throw new ClientException("対象アクティビティが存在しません。");
		}

		Activity activity = db.queryForObject("SELECT * FROM isubnb.activity WHERE id = ?",
				new BeanPropertyRowMapper<Activity>(Activity.class), activityId);

		Map<String, String> result = new HashMap<>();
		result.put("photo_1", activity.getPhoto_1());
		result.put("photo_2", activity.getPhoto_2());
		result.put("photo_3", activity.getPhoto_3());
		result.put("photo_4", activity.getPhoto_4());
		result.put("photo_5", activity.getPhoto_5());

		if (imageId < 1 || imageId > 5) {
			throw new ClientException("画像IDの指定が誤っています。");
		}
		if (result.get("photo_" + imageId) == null) {
			throw new ClientException("画像が存在しません。");
		}

		return Base64.getDecoder().decode(result.get("photo_" + imageId));
	}

	@PostMapping("/reservation_activity")
	public String createActivityReservation(@RequestBody ReserveActivityRequest req) {

		if (req.getUserId() == null) {
			throw new ClientException("ユーザIDを入力してください。");
		}
		try {
			db.queryForObject("SELECT * FROM isubnb.user WHERE id = ?",
					new BeanPropertyRowMapper<User>(User.class), req.getUserId());
		} catch (EmptyResultDataAccessException e) {
			throw new ClientException("対象ユーザが存在しません。");
		}

		if (req.getActivityId() == null) {
			throw new ClientException("アクティビティIDを入力してください");
		}
		try {
			db.queryForObject("SELECT * FROM isubnb.activity WHERE id = ?",
					new BeanPropertyRowMapper<Activity>(Activity.class), req.getActivityId());
		} catch (EmptyResultDataAccessException e) {
			throw new ClientException("対象アクティビティが存在しません。");
		}

		Activity activity = db.queryForObject("SELECT * FROM isubnb.activity WHERE id = ?",
				new BeanPropertyRowMapper<Activity>(Activity.class), req.getActivityId());

		Matcher matcherNum = Pattern.compile("[0-9]+").matcher(String.valueOf(req.getNumberOfPeople()));
		if (!matcherNum.matches()) {
			throw new ClientException("人数は数値で入力してください。");
		}
		if (!(req.getNumberOfPeople() <= activity.getMaxPeopleNum())) {
			throw new ClientException("予約可能人数を超えています。");
		}

		Matcher matcherDate = Pattern.compile("^[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])$")
				.matcher(req.getDate());
		if (!matcherDate.matches()) {
			throw new ClientException("日付はyyyy-mm-dd形式で入力してください。");
		}

		DateTimeFormatter formatter = DateTimeFormatter.ISO_LOCAL_DATE;
		LocalDate activityDate = LocalDate.parse(req.getDate(), formatter);

		List<Map<String, Object>> reserved_activity_list = db.queryForList("SELECT * FROM isubnb.activity a " +
				"JOIN isubnb.reservation_activity ra ON a.id=ra.activity_id WHERE a.id = ? AND ra.date = ?" +
				"AND rh.is_deleted = ?",
				req.getActivityId(), req.getDate(), 0);
		if (reserved_activity_list.size() != 0) {
			throw new ClientException("既に予約が入っているため、予約できません。");
		}

		String reserve_id = String.valueOf(UUID.randomUUID());
		db.update("INSERT INTO isubnb.reservation_activity(id, user_id, activity_id, date, " +
				"number_of_people, is_deleted) VALUES (?, ?, ?, ?, ?, false)",
				reserve_id, req.getUserId(), req.getActivityId(), activityDate, req.getNumberOfPeople());

		return "{\"result\" : true}";
	}

	@GetMapping("/user/{userId}/reservation_activity")
	public ReservationActivityResponse getActivityReservations(@PathVariable int userId) {

		try {
			db.queryForObject("SELECT * FROM isubnb.user WHERE id = ?",
					new BeanPropertyRowMapper<User>(User.class), userId);
		} catch (EmptyResultDataAccessException e) {
			throw new ClientException("対象ユーザが存在しません。");
		}

		List<Map<String, Object>> getList = db.queryForList(
				"SELECT DISTINCT ra.id as reservation_id, ra.number_of_people, ra.activity_id " +
						"FROM isubnb.user u " +
						"JOIN isubnb.reservation_activity ra ON u.id = ra.user_id " +
						"WHERE u.id = ? " +
						"AND ra.is_deleted = ?",
				userId, 0);

		List<ReserveActivityDetail> reserveActivityList = new ArrayList<>();
		for (Map<String, Object> map : getList) {
			Activity activity = db.queryForObject("SELECT * FROM isubnb.activity WHERE id = ?",
					new BeanPropertyRowMapper<Activity>(Activity.class), map.get("activity_id"));
			convertToResponseActivity(activity);

			String reserveId = (String) map.get("reservation_id");
			int numberOfPeople = (int) map.get("number_of_people");

			LocalDate startDate = db.queryForObject(
					"SELECT min(date) FROM isubnb.reservation_activity WHERE id = '" + reserveId + "'",
					LocalDate.class);

			LocalDate endDate = db.queryForObject(
					"SELECT max(date) FROM isubnb.reservation_activity WHERE id = '" + reserveId + "'",
					LocalDate.class);
			endDate.plusDays(1);

			reserveActivityList.add(new ReserveActivityDetail(
					reserveId, startDate, numberOfPeople,
					activity));
		}
		return new ReservationActivityResponse(reserveActivityList);
	}

	@DeleteMapping("/reservation_activity/{reserveId}")
	public String deleteActivityReservation(@PathVariable String reserveId) {
		List<ReserveActivity> reservationActivities = db.query(
				"SELECT * FROM isubnb.reservation_activity WHERE id = ? AND is_deleted = ?",
				new BeanPropertyRowMapper<ReserveActivity>(ReserveActivity.class), reserveId, 0);
		if (reservationActivities.size() == 0) {
			throw new ClientException("対象の予約が存在しませんでした。");
		}
		db.update("UPDATE isubnb.reservation_activity SET is_deleted = ? WHERE id = ?", 1, reserveId);
		return "{\"result\" : true}";
	}

	@PostMapping(value = "/homes")
	public String postHomes(@RequestParam("homes.csv") MultipartFile file) throws IOException {

		if (file == null) {
			throw new ClientException("正しい名前のCSVファイルを送信してください。");
		}

		BufferedReader br = new BufferedReader(new InputStreamReader(file.getInputStream(), StandardCharsets.UTF_8));

		String line;
		int count = 0;
		while ((line = br.readLine()) != null) {
			final String[] splitCsv = line.split(",");

			db.update("INSERT INTO isubnb.home VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
					splitCsv[0], splitCsv[1], splitCsv[2], splitCsv[3], splitCsv[4], splitCsv[5], splitCsv[6],
					splitCsv[7], splitCsv[8], splitCsv[9], splitCsv[10], splitCsv[11], splitCsv[12], splitCsv[13],
					splitCsv[14], splitCsv[15], splitCsv[16]);
			count++;
		}

		return "{\"count\": " + count + "}";
	}

	public void executeScript(String file) throws SQLException {
		Resource resource = resourceLoader.getResource("classpath:" + file);
		Connection conn = DataSourceUtils.getConnection(db.getDataSource());

		ResourceDatabasePopulator rdp = new ResourceDatabasePopulator();
		rdp.addScript(resource);
		rdp.setSqlScriptEncoding("UTF-8");
		rdp.setIgnoreFailedDrops(true);
		rdp.setContinueOnError(false);
		rdp.populate(conn);
		conn.close();
	}

	private Home convertToResponseHome(Home home) {
		String imagePath = "/api/v1/home/" + home.getId() + "/image/";
		home.setPhoto_1(imagePath + "1");
		home.setPhoto_2(imagePath + "2");
		home.setPhoto_3(imagePath + "3");
		home.setPhoto_4(imagePath + "4");
		home.setPhoto_5(imagePath + "5");
		return home;
	}

	private Activity convertToResponseActivity(Activity activity) {
		String imagePath = "/api/v1/activity/" + activity.getId() + "/image/";
		activity.setPhoto_1(imagePath + "1");
		activity.setPhoto_2(imagePath + "2");
		activity.setPhoto_3(imagePath + "3");
		activity.setPhoto_4(imagePath + "4");
		activity.setPhoto_5(imagePath + "5");
		return activity;
	}

	@ExceptionHandler(ClientException.class)
	private ResponseEntity<String> clientErrorResponse(ClientException e) {
		e.printStackTrace();
		final HttpHeaders httpHeaders = new HttpHeaders();
		final HttpStatus status = HttpStatus.BAD_REQUEST;
		httpHeaders.setContentType(MediaType.APPLICATION_JSON);
		return new ResponseEntity<String>("{\"message\": \"" + e.getMessage() + "\"}", httpHeaders, status);
	}

	public static void main(String[] args) {
		SpringApplication.run(IsubnbApp.class, args);
	}

}
