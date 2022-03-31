import csv
import io
import base64
import flask
import os
import re
import mysql.connector
import json
import logging
import datetime
import uuid

app = flask.Flask(__name__)
app.config['JSON_AS_ASCII'] = False

gunicorn_error_logger = logging.getLogger('gunicorn.error')
app.logger.handlers.extend(gunicorn_error_logger.handlers)
app.logger.setLevel(logging.DEBUG)

csv.field_size_limit(5000000000)


@app.after_request
def after_request(response):
    response.headers.add('Access-Control-Allow-Origin', '*')
    response.headers.add('Access-Control-Allow-Headers',
                         'Content-Type,Authorization')
    response.headers.add('Access-Control-Allow-Methods',
                         'GET,PUT,POST,DELETE,OPTIONS')
    return response


config = {
    'host': os.environ.get('MYSQL_HOST', '127.0.0.1'),
    'database': os.environ.get('MYSQL_DBNAME', 'isubnb'),
    'port': int(os.environ.get('MYSQL_PORT', '3306')),
    'user': os.environ.get('MYSQL_USER', 'isucon'),
    'password': os.environ.get('MYSQL_PASS', 'isucon'),
}


def dbh():
    if hasattr(flask.g, 'db'):
        return flask.g.db
    flask.g.db = mysql.connector.connect(
        **config, charset='utf8', autocommit=True, allow_local_infile=True)
    return flask.g.db


@app.teardown_appcontext
def teardown(error):
    if hasattr(flask.g, "db"):
        flask.g.db.close()


@app.errorhandler(400)
def error_400(e):
    app.logger.error('httpステータス:{}, メッセージ:{}, 詳細:{}'.format(
        e.code, e.name, e.description))
    return flask.jsonify({'message': e.description}), 400, {'Content-Type': 'application/json; charset=utf-8'}


@app.errorhandler(404)
def error_404(e):
    app.logger.error('httpステータス:{}, メッセージ:{}, 詳細:{}'.format(
        e.code, e.name, e.description))
    return flask.jsonify({'message': e.description}), 404, {'Content-Type': 'application/json; charset=utf-8'}


@app.errorhandler(500)
def error_500(e):
    app.logger.error('httpステータス:{}, メッセージ:{}, 詳細:{}'.format(
        e.code, e.name, e.description))
    return flask.jsonify({'message': e.description}), 500, {'Content-Type': 'application/json; charset=utf-8'}


@app.route('/api/v1/initialize', methods=['post'])
def initialize():
    exec_sql_file('/home/isucon/isubnb/webapp/backend/mysql/db/0_Schema.sql')
    exec_sql_file(
        '/home/isucon/isubnb/webapp/backend/mysql/db/1_CsvDataImport.sql')
    initialize_request = flask.request.json

    cur = dbh().cursor()
    cur.execute('INSERT INTO isubnb.config VALUES (%s)',
                (initialize_request['reservable_days'],))
    return flask.jsonify({'language': 'python'}), 200, {'Content-Type': 'application/json; charset=utf-8'}


@app.route('/api/v1/homes', methods=['get'])
def get_homes():
    params = flask.request.args
    location = params.get('location')
    start_date = params.get('start_date')
    end_date = params.get('end_date')
    number_of_people = params.get('number_of_people')
    style = params.get('style')

    homes = []
    cur = dbh().cursor()
    cur.execute('SELECT * FROM isubnb.home ORDER BY rate DESC, price ASC, name ASC')
    for row in cur.fetchall():
        home = {
            'id': str(row[0]),
            'name': str(row[1]),
            'address': str(row[2]),
            'location': str(row[3]),
            'max_people_num': row[4],
            'description': str(row[5]),
            'catch_phrase': str(row[6]),
            'attribute': str(row[7]),
            'style': str(row[8]),
            'price': row[9],
            'photo_1': '/api/v1/home/' + str(row[0]) + '/image/1',
            'photo_2': '/api/v1/home/' + str(row[0]) + '/image/2',
            'photo_3': '/api/v1/home/' + str(row[0]) + '/image/3',
            'photo_4': '/api/v1/home/' + str(row[0]) + '/image/4',
            'photo_5': '/api/v1/home/' + str(row[0]) + '/image/5',
            'rate': row[15],
            'owner_id': str(row[16])
        }
        homes.append(home)

    if start_date is not None and end_date is not None:
        matched_homes = []
        for home in homes:
            cur.execute('SELECT * FROM isubnb.reservation_home'
                        + ' WHERE home_id = %s'
                          ' AND %s <= date'
                          ' AND date < %s',
                        (home['id'], start_date, end_date))
            if len(cur.fetchall()) == 0:
                matched_homes.append(home)
        homes = matched_homes
    if location is not None:
        homes = list(
            filter(lambda x: x['location'] == location, homes))
    if style is not None:
        homes = list(
            filter(lambda x: x['style'] == style, homes))
    if number_of_people is not None:
        homes = list(
            filter(lambda x: x['max_people_num'] >= int(number_of_people), homes))
    result = {'count': len(homes), 'homes': homes}
    return json.dumps(result, ensure_ascii=False), 200, {'Content-Type': 'application/json; charset=utf-8'}


@app.route('/api/v1/home/<home_id>', methods=['get'])
def get_home(home_id):
    cur = dbh().cursor()

    # 宿確認
    cur.execute('SELECT * FROM isubnb.home WHERE id = %s', (home_id,))
    home_list = cur.fetchall()
    if not len(home_list) == 1:
        flask.abort(400, description='対象宿が存在しません。')

    cur.execute('SELECT * FROM isubnb.home WHERE id = %s',
                (home_id,))
    home = {}
    for row in cur.fetchall():
        home = {
            'id': str(row[0]),
            'name': str(row[1]),
            'address': str(row[2]),
            'location': str(row[3]),
            'max_people_num': row[4],
            'description': str(row[5]),
            'catch_phrase': str(row[6]),
            'attribute': str(row[7]),
            'style': str(row[8]),
            'price': row[9],
            'photo_1': '/api/v1/home/' + str(row[0]) + '/image/1',
            'photo_2': '/api/v1/home/' + str(row[0]) + '/image/2',
            'photo_3': '/api/v1/home/' + str(row[0]) + '/image/3',
            'photo_4': '/api/v1/home/' + str(row[0]) + '/image/4',
            'photo_5': '/api/v1/home/' + str(row[0]) + '/image/5',
            'rate': row[15],
            'owner_id': str(row[16])
        }
    return json.dumps(home, ensure_ascii=False), 200, {'Content-Type': 'application/json; charset=utf-8'}


@app.route('/api/v1/home/<home_id>/image/<image_id>', methods=['get'])
def get_home_image(home_id, image_id):
    cur = dbh().cursor()
    cur.execute('SELECT * FROM isubnb.home WHERE id = %s',
                (home_id,))
    home_list = cur.fetchall()
    if not len(home_list) == 1:
        flask.abort(400, description='対象宿が存在しません。')

    result = {}
    for row in home_list:
        result = {
            'photo_1': row[10],
            'photo_2': row[11],
            'photo_3': row[12],
            'photo_4': row[13],
            'photo_5': row[14],
        }

    if is_integer(image_id) is False or int(image_id) < 1 or int(image_id) > 5:
        flask.abort(400, description='画像IDの指定が誤っています。')

    if result['photo_' + image_id] is None:
        flask.abort(404, description='画像が存在しません。')

    return base64.b64decode(result['photo_' + image_id]), 200, {'Content-Type': 'image/jpeg'}


@ app.route('/api/v1/home/<home_id>/calendar', methods=['get'])
def get_home_calendar(home_id):
    cur = dbh().cursor()
    base_date = datetime.date.today()
    calendar_list = []

    cur.execute('SELECT * FROM isubnb.config')
    for row in cur.fetchall():
        reservable_days = int(row[0])
    if reservable_days == 0:
        flask.abort(400, description='予約可能日数が0日です。')

    # 宿確認
    cur.execute('SELECT * FROM isubnb.home WHERE id = %s', (home_id,))
    home_list = cur.fetchall()
    if not len(home_list) == 1:
        flask.abort(400, description='対象宿が存在しません。')

    for d in date_range(base_date, base_date + datetime.timedelta(days=reservable_days)):
        cur.execute('SELECT * FROM isubnb.reservation_home'
                    ' WHERE home_id = %s'
                    ' AND DATE(date) = %s'
                    ' AND is_deleted = %s'
                    ' ORDER BY user_id, home_id',
                    (home_id, d.strftime('%Y-%m-%d'), 0,))
        reserve_list = cur.fetchall()
        calendar = {
            'date': d.strftime('%Y-%m-%d'),
            'available': False if len(reserve_list) > 0 else True
        }
        calendar_list.append(calendar)

    return flask.jsonify({'items': calendar_list}), 200, {'Content-Type': 'application/json; charset=utf-8'}


@app.route('/api/v1/reservation_home', methods=['post'])
def post_reservation_home():
    reserve_request = flask.request.json

    cur = dbh().cursor()
    # ユーザ確認
    if not reserve_request['user_id']:
        flask.abort(400, description='ユーザIDを入力してください。')

    cur.execute('SELECT * FROM isubnb.user WHERE id = %s',
                (reserve_request['user_id'],))
    user_list = cur.fetchall()
    if not len(user_list) == 1:
        flask.abort(400, description='対象ユーザが存在しません。')

    if not reserve_request['home_id']:
        flask.abort(400, description='宿IDを入力してください。')

    cur.execute('SELECT * FROM isubnb.home WHERE id = %s',
                (reserve_request['home_id'],))
    home_list = cur.fetchall()
    if not len(home_list) == 1:
        flask.abort(400, description='対象宿が存在しません。')

    cur.execute('SELECT * FROM isubnb.home WHERE id = %s',
                (reserve_request['home_id'],))
    home = {}
    for row in cur.fetchall():
        home = {
            'id': str(row[0]),
            'name': str(row[1]),
            'address': str(row[2]),
            'location': str(row[3]),
            'max_people_num': row[4],
            'description': str(row[5]),
            'catch_phrase': str(row[6]),
            'attribute': str(row[7]),
            'style': str(row[8]),
            'price': row[9],
            'photo_1': '/api/v1/home/' + str(row[0]) + '/image/1',
            'photo_2': '/api/v1/home/' + str(row[0]) + '/image/2',
            'photo_3': '/api/v1/home/' + str(row[0]) + '/image/3',
            'photo_4': '/api/v1/home/' + str(row[0]) + '/image/4',
            'photo_5': '/api/v1/home/' + str(row[0]) + '/image/5',
            'rate': row[15],
            'owner_id': str(row[16])
        }

    if not is_integer(reserve_request['number_of_people']):
        flask.abort(400, description='人数は数値で入力してください。')

    if not int(reserve_request['number_of_people']) <= home['max_people_num']:
        flask.abort(400, description='予約可能人数を超えています。')

    if not re.match(r"^[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])$",
                    reserve_request['start_date']) or not re.match(
            r"^[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])$", reserve_request['end_date']):
        flask.abort(400, description='日付はyyyy-mm-dd形式で入力してください。')

    start_date = datetime.datetime.strptime(
        reserve_request['start_date'], '%Y-%m-%d')
    end_date = datetime.datetime.strptime(
        reserve_request['end_date'], '%Y-%m-%d')
    if not (end_date - start_date).days >= 1:
        flask.abort(400, description='日付間隔を1日以上にしてください。')

    # 予約確認
    cur.execute('SELECT * FROM isubnb.home h JOIN isubnb.reservation_home rh ON h.id=rh.home_id'
                ' WHERE h.id = %s'
                ' AND %s <= rh.date'
                ' AND rh.date < %s',
                ' AND rh.is_deleted = %s',
                (reserve_request['home_id'], reserve_request['start_date'], reserve_request['end_date'], 0))
    reserved_home_list = cur.fetchall()
    if len(reserved_home_list) != 0:
        flask.abort(400, description='既に予約が入っているため、予約できません。')

    # 予約
    reserve_id = str(uuid.uuid4())
    for num in range((end_date - start_date).days):
        cur.execute('INSERT INTO isubnb.reservation_home(id, user_id, home_id, date, number_of_people, is_deleted)'
                    ' VALUES ("%s", "%s", "%s", "%s", "%s", false)' %
                    (reserve_id,
                     reserve_request['user_id'],
                     reserve_request['home_id'],
                     start_date + datetime.timedelta(days=num),
                     reserve_request['number_of_people']
                     ))

    return flask.jsonify({'result': True}), 200, {'Content-Type': 'application/json; charset=utf-8'}


@app.route('/api/v1/activities', methods=['get'])
def get_activities():
    cur = dbh().cursor()
    params = flask.request.args
    location = params.get('location')
    date = params.get('date')

    activities = []
    cur.execute('SELECT * FROM isubnb.activity ORDER BY rate DESC, price ASC, name ASC')
    for row in cur.fetchall():
        activity = {
            'id': str(row[0]),
            'name': str(row[1]),
            'address': str(row[2]),
            'location': str(row[3]),
            'max_people_num': row[4],
            'description': str(row[5]),
            'catch_phrase': str(row[6]),
            'attribute': str(row[7]),
            'category': str(row[8]),
            'price': row[9],
            'photo_1': '/api/v1/activity/' + str(row[0]) + '/image/1',
            'photo_2': '/api/v1/activity/' + str(row[0]) + '/image/2',
            'photo_3': '/api/v1/activity/' + str(row[0]) + '/image/3',
            'photo_4': '/api/v1/activity/' + str(row[0]) + '/image/4',
            'photo_5': '/api/v1/activity/' + str(row[0]) + '/image/5',
            'rate': row[15],
            'owner_id': str(row[16])
        }
        activities.append(activity)

    if location is not None:
        activities = list(
            filter(lambda x: x['location'] == location, activities))

    if date is not None:
        matched_activities = []
        for activity in activities:
            cur.execute('SELECT * FROM isubnb.reservation_activity'
                        + ' WHERE activity_id = %s'
                          ' AND date = %s',
                        (activity['id'], date))
            if len(cur.fetchall()) == 0:
                matched_activities.append(activity)
        activities = matched_activities

    result = {'count': len(activities), 'activities': activities}
    return json.dumps(result, ensure_ascii=False), 200, {'Content-Type': 'application/json; charset=utf-8'}


@app.route('/api/v1/activity/<activity_id>', methods=['get'])
def get_activity(activity_id):
    cur = dbh().cursor()

    # アクティビティ確認
    cur.execute('SELECT * FROM isubnb.activity WHERE id = %s', (activity_id,))
    activity_list = cur.fetchall()
    if not len(activity_list) == 1:
        flask.abort(400, description='対象アクティビティが存在しません。')

    cur.execute('SELECT * FROM isubnb.activity WHERE id = %s',
                (activity_id,))
    activity = {}
    for row in cur.fetchall():
        activity = {
            'id': str(row[0]),
            'name': str(row[1]),
            'address': str(row[2]),
            'location': str(row[3]),
            'max_people_num': row[4],
            'description': str(row[5]),
            'catch_phrase': str(row[6]),
            'attribute': str(row[7]),
            'category': str(row[8]),
            'price': row[9],
            'photo_1': '/api/v1/activity/' + str(row[0]) + '/image/1',
            'photo_2': '/api/v1/activity/' + str(row[0]) + '/image/2',
            'photo_3': '/api/v1/activity/' + str(row[0]) + '/image/3',
            'photo_4': '/api/v1/activity/' + str(row[0]) + '/image/4',
            'photo_5': '/api/v1/activity/' + str(row[0]) + '/image/5',
            'rate': row[15],
            'owner_id': str(row[16])
        }
    return json.dumps(activity, ensure_ascii=False), 200, {'Content-Type': 'application/json; charset=utf-8'}


@app.route('/api/v1/activity/<activity_id>/image/<image_id>', methods=['get'])
def get_activity_image(activity_id, image_id):
    cur = dbh().cursor()
    cur.execute('SELECT * FROM isubnb.activity WHERE id = %s',
                (activity_id,))
    activity_list = cur.fetchall()
    if not len(activity_list) == 1:
        flask.abort(400, description='対象アクティビティが存在しません。')

    result = {}
    for row in activity_list:
        result = {
            'photo_1': row[10],
            'photo_2': row[11],
            'photo_3': row[12],
            'photo_4': row[13],
            'photo_5': row[14],
        }

    if is_integer(image_id) is False or int(image_id) < 1 or int(image_id) > 5:
        flask.abort(400, description='画像IDの指定が誤っています。')

    if result['photo_' + image_id] is None:
        flask.abort(404, description='画像が存在しません。')

    return base64.b64decode(result['photo_' + image_id]), 200, {'Content-Type': 'image/jpeg'}


@app.route('/api/v1/reservation_activity', methods=['post'])
def post_reservation_activity():
    reserve_request = flask.request.json

    cur = dbh().cursor()
    # ユーザ確認
    if not reserve_request['user_id']:
        flask.abort(400, description='ユーザIDを入力してください。')

    cur.execute('SELECT * FROM isubnb.user WHERE id = %s',
                (reserve_request['user_id'],))
    user_list = cur.fetchall()
    if not len(user_list) == 1:
        flask.abort(400, description='対象ユーザが存在しません。')

    # アクティビティ確認
    if not reserve_request['activity_id']:
        flask.abort(400, description='アクティビティIDを入力してください。')

    cur.execute('SELECT * FROM isubnb.activity WHERE id = %s',
                (reserve_request['activity_id'],))
    activity_list = cur.fetchall()
    if not len(activity_list) == 1:
        flask.abort(400, description='対象アクティビティが存在しません。')

    cur.execute('SELECT * FROM isubnb.activity WHERE id = %s',
                (reserve_request['activity_id'],))

    activity = {}
    for row in cur.fetchall():
        activity = {
            'id': str(row[0]),
            'name': str(row[1]),
            'address': str(row[2]),
            'location': str(row[3]),
            'max_people_num': row[4],
            'description': str(row[5]),
            'catch_phrase': str(row[6]),
            'attribute': str(row[7]),
            'category': str(row[8]),
            'price': row[9],
            'photo_1': '/api/v1/activity/' + str(row[0]) + '/image/1',
            'photo_2': '/api/v1/activity/' + str(row[0]) + '/image/2',
            'photo_3': '/api/v1/activity/' + str(row[0]) + '/image/3',
            'photo_4': '/api/v1/activity/' + str(row[0]) + '/image/4',
            'photo_5': '/api/v1/activity/' + str(row[0]) + '/image/5',
            'rate': row[15],
            'owner_id': str(row[16])
        }

    if not is_integer(reserve_request['number_of_people']):
        flask.abort(400, description='人数は数値で入力してください。')

    if not int(reserve_request['number_of_people']) <= activity['max_people_num']:
        flask.abort(400, description='予約可能人数を超えています。')

    if not re.match(r"^[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])$", reserve_request['date']):
        flask.abort(400, description='日付はyyyy-mm-dd形式で入力してください。')

    activity_date = datetime.datetime.strptime(
        reserve_request['date'], '%Y-%m-%d')

    # 予約確認
    cur.execute('SELECT * FROM isubnb.activity a JOIN isubnb.reservation_activity ra ON a.id = ra.activity_id'
                ' WHERE a.id = %s'
                ' AND ra.date = %s',
                ' AND ra.is_deleted = %s',
                (reserve_request['activity_id'], reserve_request['date']))
    reserved_activity_list = cur.fetchall()
    if len(reserved_activity_list) != 0:
        flask.abort(400, description='既に予約が入っているため、予約できません。')

    # 予約
    reserve_id = str(uuid.uuid4())
    cur.execute('INSERT INTO isubnb.reservation_activity(id, user_id, activity_id, date, number_of_people, is_deleted)'
                ' VALUES ("%s", "%s", "%s", "%s", "%s", false)' %
                (reserve_id,
                 reserve_request['user_id'],
                 reserve_request['activity_id'],
                 activity_date,
                 reserve_request['number_of_people']
                 ))

    return flask.jsonify({'result': True}), 200, {'Content-Type': 'application/json; charset=utf-8'}


@ app.route('/api/v1/user/<user_id>/reservation_home', methods=['get'])
def get_user_reservation_home(user_id):
    cur = dbh().cursor()
    reservation_list = []

    cur.execute('SELECT * FROM isubnb.user WHERE id = %s',
                (user_id,))
    user_list = cur.fetchall()
    if not len(user_list) == 1:
        flask.abort(400, description='対象ユーザが存在しません。')

    cur.execute('SELECT DISTINCT rh.id as reservation_id, rh.number_of_people, rh.home_id'
                ' FROM isubnb.user u'
                ' JOIN isubnb.reservation_home rh ON u.id = rh.user_id'
                ' WHERE u.id = %s'
                ' AND rh.is_deleted = %s',
                (user_id, 0))

    for row in cur.fetchall():
        cur.execute('SELECT * FROM isubnb.home WHERE id = %s',
                    (row[2],))
        for row_home in cur.fetchall():
            home = {
                'id': str(row_home[0]),
                'name': str(row_home[1]),
                'address': str(row_home[2]),
                'location': str(row_home[3]),
                'max_people_num': row_home[4],
                'description': str(row_home[5]),
                'catch_phrase': str(row_home[6]),
                'attribute': str(row_home[7]),
                'style': str(row_home[8]),
                'price': row_home[9],
                'photo_1': '/api/v1/home/' + str(row_home[0]) + '/image/1',
                'photo_2': '/api/v1/home/' + str(row_home[0]) + '/image/2',
                'photo_3': '/api/v1/home/' + str(row_home[0]) + '/image/3',
                'photo_4': '/api/v1/home/' + str(row_home[0]) + '/image/4',
                'photo_5': '/api/v1/home/' + str(row_home[0]) + '/image/5',
                'rate': row_home[15],
                'owner_id': str(row_home[16])
            }

        reserve_id = str(row[0])
        number_of_people = row[1]

        start_date = ""
        cur.execute(
            'SELECT min(date) FROM isubnb.reservation_home WHERE id = %s', (reserve_id,))
        for row in cur.fetchall():
            start_date = str(row[0])

        end_date = ""
        cur.execute(
            'SELECT max(date) FROM isubnb.reservation_home WHERE id = %s', (reserve_id,))
        for row in cur.fetchall():
            end_date_datetime = datetime.datetime.strptime(str(row[0]), '%Y-%m-%d') + datetime.timedelta(days=1)
            end_date = end_date_datetime.strftime('%Y-%m-%d')

        reservation = {
            'reserve_id': reserve_id,
            "start_date": start_date,
            "end_date": end_date,
            'number_of_people': number_of_people,
            'reserve_home': home
        }
        reservation_list.append(reservation)

    return flask.jsonify({'reservations': reservation_list}), 200, {'Content-Type': 'application/json; charset=utf-8'}


@ app.route('/api/v1/user/<user_id>/reservation_activity', methods=['get'])
def get_user_reservation_activity(user_id):
    cur = dbh().cursor()
    reservation_list = []

    cur.execute('SELECT * FROM isubnb.user WHERE id = %s',
                (user_id,))
    user_list = cur.fetchall()
    if not len(user_list) == 1:
        flask.abort(400, description='対象ユーザが存在しません。')

    cur.execute('SELECT DISTINCT ra.id as reservation_id, ra.number_of_people, ra.activity_id'
                ' FROM isubnb.user u'
                ' JOIN isubnb.reservation_activity ra ON u.id = ra.user_id'
                ' WHERE u.id = %s'
                ' AND ra.is_deleted = %s',
                (user_id, 0))

    for row in cur.fetchall():
        cur.execute('SELECT * FROM isubnb.activity WHERE id = %s',
                    (row[2],))
        for row_activity in cur.fetchall():
            activity = {
                'id': str(row_activity[0]),
                'name': str(row_activity[1]),
                'address': str(row_activity[2]),
                'location': str(row_activity[3]),
                'max_people_num': row_activity[4],
                'description': str(row_activity[5]),
                'catch_phrase': str(row_activity[6]),
                'attribute': str(row_activity[7]),
                'category': str(row_activity[8]),
                'price': row_activity[9],
                'photo_1': '/api/v1/activity/' + str(row_activity[0]) + '/image/1',
                'photo_2': '/api/v1/activity/' + str(row_activity[0]) + '/image/2',
                'photo_3': '/api/v1/activity/' + str(row_activity[0]) + '/image/3',
                'photo_4': '/api/v1/activity/' + str(row_activity[0]) + '/image/4',
                'photo_5': '/api/v1/activity/' + str(row_activity[0]) + '/image/5',
                'rate': row_activity[15],
                'owner_id': str(row_activity[16])
            }

        reserve_id = str(row[0])
        number_of_people = row[1]

        start_date = ""
        cur.execute(
            'SELECT min(date) FROM isubnb.reservation_activity WHERE id = %s', (reserve_id,))
        for row in cur.fetchall():
            start_date = str(row[0])

        end_date = ""
        cur.execute(
            'SELECT max(date) FROM isubnb.reservation_activity WHERE id = %s', (reserve_id,))
        for row in cur.fetchall():
            end_date_datetime = datetime.datetime.strptime(str(row[0]), '%Y-%m-%d') + datetime.timedelta(days=1)
            end_date = end_date_datetime.strftime('%Y-%m-%d')

        reservation = {
            'reserve_id': reserve_id,
            "reserve_date": start_date,
            'number_of_people': number_of_people,
            'reserve_activity': activity
        }
        reservation_list.append(reservation)

    return flask.jsonify({'reservations': reservation_list}), 200, {'Content-Type': 'application/json; charset=utf-8'}


@ app.route('/api/v1/reservation_home/<reserve_id>', methods=['delete'])
def delete_reservation_home(reserve_id):
    cur = dbh().cursor()
    cur.execute('SELECT * FROM isubnb.reservation_home WHERE id = %s AND is_deleted = %s',
                (reserve_id, 0))
    if len(cur.fetchall()) == 0:
        flask.abort(400, description='対象の予約が存在しませんでした。')

    cur.execute('UPDATE isubnb.reservation_home SET is_deleted = %s'
                ' WHERE id = %s',
                (1, reserve_id))

    return flask.jsonify({'result': True}), 200, {'Content-Type': 'application/json; charset=utf-8'}


@ app.route('/api/v1/reservation_activity/<reserve_id>', methods=['delete'])
def delete_reservation_activity(reserve_id):
    cur = dbh().cursor()
    cur.execute('SELECT * FROM isubnb.reservation_activity WHERE id = %s AND is_deleted = %s',
                (reserve_id, 0))
    if len(cur.fetchall()) == 0:
        flask.abort(400, description='対象の予約が存在しませんでした。')

    cur.execute('UPDATE isubnb.reservation_activity SET is_deleted = %s'
                ' WHERE id = %s',
                (1, reserve_id))

    return flask.jsonify({'result': True}), 200, {'Content-Type': 'application/json; charset=utf-8'}


@app.route('/api/v1/homes', methods=['post'])
def post_homes():
    requestCSV = flask.request.files.get('homes.csv')
    if requestCSV is None:
        return flask.abort(400, description='正しい名前のCSVファイルを送信してください。')

    path = '/home/isucon/isubnb/webapp/backend/mysql/data/'
    requestCSV.save(os.path.join(path, requestCSV.filename))
    f = open(path + requestCSV.filename, 'r')
    csv_lines = csv.reader(f)

    count = 0
    cur = dbh().cursor()
    for row in csv_lines:
        cur.execute(
            'INSERT INTO isubnb.home VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s)',
            (row[0], row[1], row[2], row[3], row[4], row[5], row[6], row[7], row[8], row[9],
             row[10], row[11], row[12], row[13], row[14], row[15], row[16]))
        count += 1
    return flask.jsonify({'count': count}), 200, {'Content-Type': 'application/json; charset=utf-8'}


def exec_sql_file(sql_file):
    cur = dbh().cursor()
    statement = ""
    for line in open(sql_file):
        if re.match(r'--', line):
            continue
        if not re.search(r'[^-;]+;', line):
            statement = statement + line
        else:
            statement = statement + line
            cur.execute(statement)
            statement = ""
    cur.close()


def is_integer(n):
    try:
        float(n)
    except ValueError:
        return False
    else:
        return float(n).is_integer()


def date_range(start, stop, step=datetime.timedelta(1)):
    current = start
    while current < stop:
        yield current
        current += step


if __name__ == "__main__":
    app.run(debug=True, threaded=True)
